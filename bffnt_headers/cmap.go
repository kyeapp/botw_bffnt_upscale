package bffnt_headers

import (
	"encoding/binary"
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

	// key is a character's ascii code
	// value is the character's index in the tglp's SheetData
	CharacterIndexMap map[uint16]uint16
}

func (cmap *CMAP) Decode(allRaw []byte, cmapOffset uint32) {
	// CMAPOffset skips the first 8 bytes that contain the CMAP Magic Header
	headerStart := int(cmapOffset - 8)
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

	// Direct mapping is the most space efficient of mapping type. It is used
	// if all the characters in the range are to be indexed. The reserved data
	// is characterOffset. Character offset is needed if the direct map is not
	// the first map to be read. Instead of storing any additional daata other
	// than the header, bytes are saved by just storing an offset and
	// calculating the index based on the character's ascii code. With each new
	// character in a direct character map, the character's index is
	// incremented by 1. The character offset should be equal to the total
	// number of characters indexed from previous CMAPs.
	indexMap := make(map[uint16]uint16, 0)
	switch cmap.MappingMethod {
	case 0:
		characterOffset := cmap.Reserved
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charAsciiCode := i
			charIndex := i - cmap.CodeBegin + characterOffset
			indexMap[charAsciiCode] = charIndex

			// fmt.Printf("direct %#U %d\n", rune(charCode), charIdx)
		}
		cmap.CharacterIndexMap = indexMap

		characterCodeCount := int(cmap.CodeEnd - cmap.CodeBegin + 1)
		assertEqual(characterCodeCount, len(cmap.CharacterIndexMap))
		// totalBytesSoFar := int(headerStart) + CMAP_HEADER_SIZE + dataLen
		// calculatedSectionSize := CMAP_HEADER_SIZE + dataLen + paddingToNext8ByteBoundary(totalBytesSoFar)

		// calculatedSectionSize := CMAP_HEADER_SIZE + dataPos
		// fmt.Println("calc section size", calculatedSectionSize)
		// diff := int(cmap.SectionSize) - calculatedSectionSize
		// fmt.Println("diff:", diff)
		// for i := 0; i < diff; i++ {
		// 	offsetFromData := CMAP_HEADER_SIZE + dataPos
		// 	fmt.Println(offsetFromData+i, data[dataPos+i])
		// }
		// assertEqual(int(cmap.SectionSize), calculatedSectionSize)
		break

	// Table mapping is used when there are unused characters in between the
	// range of CodeBegin and CodeEnd. Character Index is stored in the next
	// (CodeEnd - CodeStart + 1) amount of bytes after the header. Unused
	// characters will have an index of MaxUint16 (65535).
	case 1:
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charAsciiCode := i
			charIndex := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			if charIndex != 65535 { // math.MaxUint16
				indexMap[charAsciiCode] = charIndex

				// fmt.Printf("table %#U %d\n", charCode, charIdx)
			}
			dataPos += 2
		}
		cmap.CharacterIndexMap = indexMap

		// calculatedSectionSize := CMAP_HEADER_SIZE + dataPos
		// fmt.Println("calc section size", calculatedSectionSize)
		// diff := int(cmap.SectionSize) - calculatedSectionSize
		// fmt.Println("diff:", diff)
		// for i := 0; i < diff; i++ {
		// 	offsetFromData := CMAP_HEADER_SIZE + dataPos
		// 	fmt.Println(offsetFromData+i, data[dataPos+i])
		// }
		// assertEqual(int(cmap.SectionSize), calculatedSectionSize)

		break

	// Scan mapping is used for individual ascii code indexing. And the most
	// space inefficient type of character mapping. It is done by storing an
	// array of ascii codes and its index after the header. The first uint16 is
	// the amount of glyphs to read. After that the bytes are read in uint16
	// pairs. Read a uint16 for the character ascii code and then another
	// uint16 for the character index.
	case 2:
		charCount := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		dataPos += 2

		for i := uint16(0); i < charCount; i++ {
			charAsciiCode := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			charIndex := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
			indexMap[charAsciiCode] = charIndex
			// fmt.Printf("table %#U %d\n", charAsciiCode, charIndex)

			dataPos += 4
		}
		cmap.CharacterIndexMap = indexMap

		// calculatedSectionSize := CMAP_HEADER_SIZE + dataPos
		// fmt.Println("calc section size", calculatedSectionSize)
		// diff := int(cmap.SectionSize) - calculatedSectionSize
		// fmt.Println("diff:", diff)
		// for i := 0; i < diff; i++ {
		// 	offsetFromData := CMAP_HEADER_SIZE + dataPos
		// 	fmt.Println(offsetFromData+i, data[dataPos+i])
		// }
		// assertEqual(int(cmap.SectionSize), calculatedSectionSize)
		break

	default:
		panic("unknown mapping method")
	}

	// fmt.Println("cmap padding start", headerStart+CMAP_HEADER_SIZE+dataPos)
	// totalBytesSoFar := int(headerStart) + CMAP_HEADER_SIZE + dataLen
	// fmt.Println("padding to next 8 byte boundary", paddingToNext8ByteBoundary(totalBytesSoFar))

	if Debug {
		dataPosEnd := headerEnd + dataPos
		fmt.Printf("Read section total of %d bytes\n", dataPosEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header           %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("data calculated  %-8d to  %d\n", headerEnd, dataPosEnd)
		padding := headerStart + int(cmap.SectionSize) - dataPosEnd
		fmt.Printf("pad %d bytes     %-8d to  %d\n", padding, dataPosEnd, dataPosEnd+padding)
		fmt.Println()
	}
}

func (cmap *CMAP) encode() []byte {
	return nil
}

func DecodeAllCmaps(allRaw []byte, offset uint32) []CMAP {
	res := make([]CMAP, 0)

	for offset != 0 {
		var currentCmap CMAP
		currentCmap.Decode(allRaw, offset)
		res = append(res, currentCmap)

		offset = currentCmap.NextCMAPOffset
	}

	return res
}
