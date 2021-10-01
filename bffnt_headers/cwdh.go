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
	StartIndex     uint16 // 0x08    0x02  Start Index
	EndIndex       uint16 // 0x0A    0x02  End Index
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

func (cwdh *CWDH) decode(raw []byte, cwdhOffset uint32) {
	// FINF.CWDHOffset skips the first 8 bytes that contain the CWDH Magic Header
	headerStart := int(cwdhOffset - 8)
	headerEnd := headerStart + CWDH_HEADER_SIZE
	headerBytes := raw[headerStart:headerEnd]
	cwdh.decodeHeader(headerBytes)

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

func (cwdh *CWDH) decodeHeader(raw []byte) {
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

func (cwdh *CWDH) encode() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(cwdh.MagicHeader))
	_ = binary.Write(w, binary.BigEndian, cwdh.SectionSize)
	_ = binary.Write(w, binary.BigEndian, cwdh.StartIndex)
	_ = binary.Write(w, binary.BigEndian, cwdh.EndIndex)
	_ = binary.Write(w, binary.BigEndian, cwdh.NextCWDHOffset)
	w.Flush()

	assertEqual(CWDH_HEADER_SIZE, len(buf.Bytes()))

	// encode cwdh data

	return buf.Bytes()
}
