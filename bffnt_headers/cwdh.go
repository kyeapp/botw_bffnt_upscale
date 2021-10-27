package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type CWDH struct { //        Offset  Size  Description
	MagicHeader    string // 0x00    0x04  Magic Header (CWDH)
	SectionSize    uint32 // 0x04    0x04  Section Size
	StartIndex     uint16 // 0x08    0x02  Start Index, typically 0?
	EndIndex       uint16 // 0x0A    0x02  End Index, number of glyphs - 1?
	NextCWDHOffset uint32 // 0x0C    0x04  Next CWDH Offset
	Glyphs         []glyphInfo

	// Data until the end of the section comes in tuples of 3 bytes
	// LeftWidth   uint8  // 0x10    0x04  Char Widths (3 bytes: Left, Glyph Width, Char Width)
	// GlyphWidth  uint8
	// CharWidth   uint8
}

type glyphInfo struct {
	LeftWidth  int8 // left spacing
	GlyphWidth uint8
	CharWidth  uint8
}

func (cwdh *CWDH) Upscale(scale float64) {
	for i, _ := range cwdh.Glyphs {
		cwdh.Glyphs[i].LeftWidth = int8(math.Ceil(float64(cwdh.Glyphs[i].LeftWidth) * scale))
		cwdh.Glyphs[i].GlyphWidth = uint8(math.Ceil(float64(cwdh.Glyphs[i].GlyphWidth) * scale))
		cwdh.Glyphs[i].CharWidth = uint8(math.Ceil(float64(cwdh.Glyphs[i].CharWidth) * scale))
	}
}

func (cwdh *CWDH) Decode(raw []byte, cwdhOffset uint32) {
	headerStart := int(cwdhOffset) - 8
	headerEnd := headerStart + CWDH_HEADER_SIZE
	headerBytes := raw[headerStart:headerEnd]
	cwdh.DecodeHeader(headerBytes)

	// Character width data is read in tuples of 3 bytes.  The glyph width info
	// is ordered corresponding to a character index.
	dataSize := int(cwdh.SectionSize - CWDH_HEADER_SIZE)
	dataStart := int(headerEnd) // data starts when the header ends
	dataEnd := dataStart + dataSize
	data := raw[dataStart:dataEnd]
	resultGlyphs := make([]glyphInfo, 0)

	dataPos := 0
	for i := int(cwdh.StartIndex); i <= int(cwdh.EndIndex); i++ {
		currentGlyph := glyphInfo{
			LeftWidth:  int8(data[dataPos]),
			GlyphWidth: uint8(data[dataPos+1]),
			CharWidth:  uint8(data[dataPos+2]),
		}
		resultGlyphs = append(resultGlyphs, currentGlyph)
		dataPos += 3
	}
	cwdh.Glyphs = resultGlyphs

	leftoverData := data[dataPos:]
	verifyLeftoverBytes(leftoverData)

	assertEqual(int(cwdh.EndIndex+1), len(cwdh.Glyphs))

	if Debug {
		dataEnd := dataStart + dataPos
		fmt.Printf("Read section total of %d bytes\n", dataEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header           %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("data calculated  %-8d to  %d\n", dataStart, dataEnd)
		fmt.Printf("leftover bytes   %-8d to  %d\n", dataEnd, dataEnd+len(leftoverData))
		fmt.Println()
	}
}

func (cwdh *CWDH) DecodeHeader(raw []byte) {
	assertEqual(CWDH_HEADER_SIZE, len(raw))

	cwdh.MagicHeader = string(raw[0:4])
	cwdh.SectionSize = binary.BigEndian.Uint32(raw[4:8])
	cwdh.StartIndex = binary.BigEndian.Uint16(raw[8:10])
	cwdh.EndIndex = binary.BigEndian.Uint16(raw[10:12])
	cwdh.NextCWDHOffset = binary.BigEndian.Uint32(raw[12:CWDH_HEADER_SIZE])

	if Debug {
		pprint(cwdh)
	}
}

func DecodeCWDHs(allRaw []byte, startingOffset uint32) []CWDH {
	res := make([]CWDH, 0)

	offset := startingOffset
	for offset != 0 {
		var currentCWDH CWDH
		currentCWDH.Decode(allRaw, offset)
		res = append(res, currentCWDH)

		offset = currentCWDH.NextCWDHOffset
	}

	return res
}

// Encodes a single cwdh.
// The start offset passed is either the starting finf.cwdhOffset or the last cwdh's NextCWDHOffset
func (cwdh *CWDH) Encode(startOffset uint32, isLastCWDH bool) []byte {
	var dataBuf bytes.Buffer
	dataWriter := bufio.NewWriter(&dataBuf)

	// encode cwdh data. We need to know the length of the raw glyph data to
	// know the section size
	for _, glyph := range cwdh.Glyphs {
		binaryWrite(dataWriter, glyph.LeftWidth)
		binaryWrite(dataWriter, glyph.GlyphWidth)
		binaryWrite(dataWriter, glyph.CharWidth)
	}
	dataWriter.Flush()

	padToNext4ByteBoundary(dataWriter, &dataBuf, int(startOffset))

	glyphData := dataBuf.Bytes()
	// Calculate and edit the header information
	cwdh.SectionSize = uint32(CWDH_HEADER_SIZE + len(glyphData))
	cwdh.StartIndex = uint16(0)
	cwdh.EndIndex = uint16(len(cwdh.Glyphs) - 1)
	if isLastCWDH {
		cwdh.NextCWDHOffset = 0
	} else {
		// CMAP is a recursive structure, the +8 bytes should have been added
		// already to make calculations easier
		cwdh.NextCWDHOffset = uint32(int(startOffset) + CWDH_HEADER_SIZE + len(glyphData))
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	// Write raw data of the header and data
	_, _ = w.Write([]byte(cwdh.MagicHeader))
	binaryWrite(w, cwdh.SectionSize)
	binaryWrite(w, cwdh.StartIndex)
	binaryWrite(w, cwdh.EndIndex)
	binaryWrite(w, cwdh.NextCWDHOffset)
	_, _ = w.Write(glyphData)
	w.Flush()

	return buf.Bytes()
}

func EncodeCWDHs(CWDHs []CWDH, finfCWDHOffset int) []byte {
	res := make([]byte, 0)

	offset := uint32(finfCWDHOffset)
	for i, currentCWDH := range CWDHs {
		isLast := false
		if i == len(CWDHs)-1 {
			isLast = true
		}

		cwdhBytes := currentCWDH.Encode(offset, isLast)

		res = append(res, cwdhBytes...)
		offset = currentCWDH.NextCWDHOffset
	}

	return res
}

// takes a cwdh list and adds the section size together.
func totalCwdhSectionSize(cwdhList []CWDH) (totalSectionSize int) {
	totalSectionSize = 0

	for _, currentCWDH := range cwdhList {
		totalSectionSize += int(currentCWDH.SectionSize)
	}

	return totalSectionSize
}
