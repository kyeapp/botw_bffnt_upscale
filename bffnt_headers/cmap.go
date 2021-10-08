package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// A single cmap contains information about a character's texture location in
// the images. All cmaps must be decoded to have all character indexes. The
// different mapping methods exists to save as much bytes as possible.
type CMAP struct { //         Offset  Size  Description
	MagicHeader    string // 0x00    0x04  Magic Header (CMAP)
	SectionSize    uint32 // 0x04    0x04  Section Size
	CodeBegin      uint16 // 0x08    0x02  Code Begin
	CodeEnd        uint16 // 0x0A    0x02  Code End
	MappingMethod  uint16 // 0x0C    0x02  Mapping Method (0 = Direct, 1 = Table, 2 = Scan)
	Reserved       uint16 // 0x0E    0x02  Reserved?
	NextCMAPOffset uint32 // 0x10    0x04  Next CMAP Offset

	CharacterOffset uint16 // used for direct maps
	// This is a pair of arrays that hold the ascii and it's index in the font
	// texture. Characters that have an index of MaxUint16 (65535) are to be ignored.
	CharAscii []uint16
	CharIndex []uint16
}

type AsciiIndexPair struct {
	CharAscii uint16
	CharIndex uint16
}

func (cmap *CMAP) Decode(allRaw []byte, cmapOffset uint32) {
	headerStart := int(cmapOffset) - 8
	headerEnd := headerStart + CMAP_HEADER_SIZE
	headerRaw := allRaw[headerStart:headerEnd]

	assertEqual(CMAP_HEADER_SIZE, len(headerRaw))

	cmap.MagicHeader = string(headerRaw[0:4])
	cmap.SectionSize = binary.BigEndian.Uint32(headerRaw[4:8])
	cmap.CodeBegin = binary.BigEndian.Uint16(headerRaw[8:10])
	cmap.CodeEnd = binary.BigEndian.Uint16(headerRaw[10:12])
	cmap.MappingMethod = binary.BigEndian.Uint16(headerRaw[12:14])
	cmap.Reserved = binary.BigEndian.Uint16(headerRaw[14:16])
	cmap.NextCMAPOffset = binary.BigEndian.Uint32(headerRaw[16:CMAP_HEADER_SIZE])

	if Debug {
		pprint(cmap)
	}

	dataEnd := headerStart + int(cmap.SectionSize)
	data := allRaw[headerEnd:dataEnd]
	dataPos := 0

	indexSlice := make([]uint16, 0)
	asciiSlice := make([]uint16, 0)
	// Direct mapping is the most space efficient of mapping type. It is used
	// if all the characters in the range are to be indexed. Character offset
	// is needed if the direct map is not the first map to be read. Instead of
	// storing any additional daata other than the header, bytes are saved by
	// just storing an offset and calculating the index based on the
	// character's ascii code. With each new character in a direct character
	// map, the character's index is incremented by 1. The character offset
	// should be equal to the total number of characters indexed from previous
	// CMAPs.
	switch cmap.MappingMethod {
	case 0:
		characterOffset := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		dataPos += 2
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charAsciiCode := i
			charIndex := i - cmap.CodeBegin + characterOffset
			asciiSlice = append(asciiSlice, charAsciiCode)
			indexSlice = append(indexSlice, charIndex)

			// fmt.Printf("direct %#U %d\n", rune(charAsciiCode), charIndex)
		}

		break

	// Table mapping is used when there are unused characters in between the
	// range of CodeBegin and CodeEnd. Character Index is stored in the next
	// (CodeEnd - CodeStart + 1) amount of bytes after the header. Unused
	// characters will have an index of MaxUint16 (65535).
	case 1:
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charAsciiCode := i
			charIndex := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			asciiSlice = append(asciiSlice, charAsciiCode)
			indexSlice = append(indexSlice, charIndex)

			// fmt.Printf("table %#U %d\n", rune(charAsciiCode), charIndex)

			dataPos += 2
		}

		break

	// Scan mapping is used for individual ascii code indexing. And the most
	// space inefficient type of character mapping. It is done by storing an
	// array of ascii codes and its index after the header. The first uint16 is
	// the amount of pairs of (glyph, index) to read. After that the bytes are
	// read in uint16 pairs. Read a uint16 for the character ascii code and
	// then another uint16 for the character index.
	case 2:
		charCount := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		dataPos += 2

		for i := uint16(0); i < charCount; i++ {
			charAsciiCode := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			charIndex := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
			asciiSlice = append(asciiSlice, charAsciiCode)
			indexSlice = append(indexSlice, charIndex)

			// fmt.Printf("individual %#U %d\n", rune(charAsciiCode), charIndex)

			dataPos += 4
		}

		break

	default:
		handleErr(errors.New("unknown mapping method"))
	}
	cmap.CharAscii = asciiSlice
	cmap.CharIndex = indexSlice

	leftoverData := data[dataPos:]
	verifyLeftoverBytes(leftoverData)
	assertEqual(len(cmap.CharAscii), len(cmap.CharIndex))

	if Debug {
		dataPosEnd := headerEnd + dataPos
		fmt.Printf("Read section total of %d bytes\n", dataPosEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header           %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("data calculated  %-8d to  %d\n", headerEnd, dataPosEnd)
		fmt.Printf("leftover bytes   %-8d to  %d\n", dataPosEnd, dataPosEnd+len(leftoverData))
		fmt.Println()
	}
}

func DecodeCMAPs(allRaw []byte, startingOffset uint32) []CMAP {
	res := make([]CMAP, 0)

	offset := startingOffset
	for offset != 0 {
		var currentCMAP CMAP
		currentCMAP.Decode(allRaw, offset)
		res = append(res, currentCMAP)

		offset = currentCMAP.NextCMAPOffset
	}

	return res
}

// Encodes a single cmap.
// The start offset passed in should be the total number of bytes written so far
func (cmap *CMAP) Encode(startOffset uint32, isLastCMAP bool) []byte {
	var cmapDataBuf bytes.Buffer
	dataWriter := bufio.NewWriter(&cmapDataBuf)

	// encode cmap data. We need to know the length of the raw glyph data to
	// know the section size
	switch cmap.MappingMethod {
	case 0:
		binaryWrite(dataWriter, cmap.CharacterOffset)
	case 1:
		for i, _ := range cmap.CharIndex {
			binaryWrite(dataWriter, cmap.CharIndex[i])
		}
	case 2:
		// first uint16 is amount of (charAscii, charIndex) pairs
		binaryWrite(dataWriter, uint16(len(cmap.CharIndex)))
		for i, _ := range cmap.CharIndex {
			binaryWrite(dataWriter, cmap.CharAscii[i])
			binaryWrite(dataWriter, cmap.CharIndex[i])
		}
	}

	// Nintendo pads to a 4 byte boundary (2 extra bytes max), but it seems like its not needed.

	dataWriter.Flush()
	cmapData := cmapDataBuf.Bytes()

	// Calculate and edit the header information
	cmap.SectionSize = uint32(CMAP_HEADER_SIZE + len(cmapData))
	// Assume the startOffset already had +8 added to it to skip the magic header
	cmap.NextCMAPOffset = uint32(int(startOffset) + CMAP_HEADER_SIZE + len(cmapData))

	if isLastCMAP {
		// terminate cmap list by setting offset to 0
		cmap.NextCMAPOffset = 0
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	// Write raw data of the header and data
	_, _ = w.Write([]byte(cmap.MagicHeader))
	binaryWrite(w, cmap.SectionSize)
	binaryWrite(w, cmap.CodeBegin)
	binaryWrite(w, cmap.CodeEnd)
	binaryWrite(w, cmap.MappingMethod)
	binaryWrite(w, cmap.Reserved)
	binaryWrite(w, cmap.NextCMAPOffset)
	_, _ = w.Write(cmapData)

	w.Flush()
	return buf.Bytes()
}

func EncodeCMAPs(CMAPs []CMAP, startingOffset int) []byte {
	res := make([]byte, 0)

	// Offset to write should have 8 bytes added to it to skip the magic header
	// since cmap is a recursive structure, all cmap's NextCMAPOffset will be
	// correctly offset by 8
	offset := uint32(startingOffset) + 8
	for i, currentCMAP := range CMAPs {
		isLast := false
		if i == len(CMAPs)-1 {
			isLast = true
		}

		cmapBytes := currentCMAP.Encode(offset, isLast)

		res = append(res, cmapBytes...)
		offset = currentCMAP.NextCMAPOffset
	}

	return res
}
