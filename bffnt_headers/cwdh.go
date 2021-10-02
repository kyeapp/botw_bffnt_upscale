package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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
	GlyphWidth int8
	CharWidth  int8
}

func (cwdh *CWDH) Decode(raw []byte, cwdhOffset uint32) {
	// FINF.CWDHOffset skips the first 8 bytes that contain the CWDH Magic Header
	headerStart := int(cwdhOffset - 8)
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
			GlyphWidth: int8(data[dataPos+1]),
			CharWidth:  int8(data[dataPos+2]),
		}
		resultGlyphs = append(resultGlyphs, currentGlyph)
		dataPos += 3
	}
	cwdh.Glyphs = resultGlyphs

	// hs := int(headerStart)
	// fmt.Println(hs)                                         // 532480
	// fmt.Println(hs + CWDH_HEADER_SIZE)                      // 532496
	// fmt.Println(hs + CWDH_HEADER_SIZE + 3*len(cwdh.Glyphs)) // 534326
	// fmt.Println(dataEnd)                                    // 534328

	totalBytesSoFar := int(headerStart) + CWDH_HEADER_SIZE + dataPos
	calculatedCWDHSectionSize := CWDH_HEADER_SIZE + dataPos + paddingToNext8ByteBoundary(totalBytesSoFar)
	assertEqual(int(cwdh.SectionSize), calculatedCWDHSectionSize)
	assertEqual(int(cwdh.EndIndex+1), len(cwdh.Glyphs))

	if Debug {
		dataPosGlobal := headerEnd + dataPos
		fmt.Printf("Read section total of %d bytes\n", dataPosGlobal-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header           %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("data calculated  %-8d to  %d\n", dataStart, dataPosGlobal)
		padding := paddingToNext8ByteBoundary(totalBytesSoFar)
		fmt.Printf("pad %d byte      %-8d to  %d\n", padding, dataPosGlobal, dataPosGlobal+padding)
		fmt.Println()
	}

	//TODO decode more than 1 cwdh
}

// After every CWDH and CMAP section and its data is encoded. There is padding
// that happens to bring the total bytes to the next 8 byte boundary. This
// includes all the bytes of CFNT, FINF, every CWDH and every CMAP that was
// written before.
func paddingToNext8ByteBoundary(dataLen int) int {
	return 8 - dataLen%8
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

// Encodes a single cwdh.
// The start offset passed in should be the total number of bytes written so far
func (cwdh *CWDH) Encode(startOffset uint32, isLastCWDH bool) []byte {
	var dataBuf bytes.Buffer
	dataWriter := bufio.NewWriter(&dataBuf)

	// encode cwdh data. We need to know the length of the raw glyph data to
	// know the section size
	for _, glyph := range cwdh.Glyphs {
		binaryWrite(dataWriter, glyph.LeftWidth)
		fmt.Println(glyph)
		binaryWrite(dataWriter, glyph.GlyphWidth)
		binaryWrite(dataWriter, glyph.CharWidth)
	}
	glyphData := dataBuf.Bytes()
	fmt.Println("glyphData:", len(glyphData))
	fmt.Println(glyphData)

	// Calculate and edit the header information
	cwdh.SectionSize = uint32(CWDH_HEADER_SIZE + len(glyphData))
	cwdh.StartIndex = uint16(0)
	cwdh.EndIndex = uint16(len(cwdh.Glyphs) - 1)
	if isLastCWDH {
		cwdh.NextCWDHOffset = 0
	} else {
		// + 8 bytes to skip the next CWDH magic header and section size
		cwdh.NextCWDHOffset = uint32(int(startOffset) + CWDH_HEADER_SIZE + len(glyphData) + 8)
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

func DecodeCWDHs(allRaw []byte, FINF_CWDH_Offset uint32) []CWDH {
	res := make([]CWDH, 0)

	offset := FINF_CWDH_Offset
	for offset != 0 {
		var currentCWDH CWDH
		currentCWDH.Decode(allRaw, offset)
		res = append(res, currentCWDH)

		offset = currentCWDH.NextCWDHOffset
	}

	return res
}

func EncodeCWDHs(CWDHs []CWDH, startingOffset int) []byte {
	res := make([]byte, 0)

	offset := uint32(startingOffset)
	for i, currentCWDH := range CWDHs {
		// Check if last CWDH. The last one will have a NextCWDHOffset of 0
		isLast := false
		if i == len(CWDHs)-1 {
			isLast = true
		}

		cwdhBytes := currentCWDH.Encode(offset, isLast)

		res = append(res, cwdhBytes...)
		offset = currentCWDH.NextCWDHOffset
	}

	// possible TODO? pad to the next 8 byte boundary

	return res
}
