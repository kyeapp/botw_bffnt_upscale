package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
)

type FINF struct { //  Offset  Size  Description
	MagicHeader       string // 0x00    0x04  Magic Header (FINF)
	SectionSize       uint32 // 0x04    0x04  Section Size
	FontType          uint8  // 0x08    0x01  Font Type
	Height            uint8  // 0x09    0x01  Height
	Width             uint8  // 0x0A    0x01  Width
	Ascent            uint8  // 0x0B    0x01  Ascent
	LineFeed          uint16 // 0x0C    0x02  Line Feed
	AlterCharIndex    uint16 // 0x0E    0x02  Alter Char Index
	DefaultLeftWidth  uint8  // 0x10    0x03  Default Width (3 bytes: Left, Glyph Width, Char Width)
	DefaultGlyphWidth uint8
	DefaultCharWidth  uint8
	Encoding          uint8  // 0x13    0x01  Encoding
	TGLPOffset        uint32 // 0x14    0x04  TGLP Offset
	CWDHOffset        uint32 // 0x18    0x04  CWDH Offset
	CMAPOffset        uint32 // 0x1C    0x04  CMAP Offset
}

// Version 4 (BFFNT)
func (finf *FINF) Decode(raw []byte) {
	headerStart := CFNT_HEADER_SIZE
	headerEnd := headerStart + FINF_HEADER_SIZE
	headerRaw := raw[headerStart:headerEnd]
	assertEqual(FINF_HEADER_SIZE, len(headerRaw))

	finf.MagicHeader = string(headerRaw[0:4])
	finf.SectionSize = binary.BigEndian.Uint32(headerRaw[4:8])
	finf.FontType = headerRaw[8] // byte == uint8
	finf.Height = headerRaw[9]
	finf.Width = headerRaw[10]
	finf.Ascent = headerRaw[11]
	finf.LineFeed = binary.BigEndian.Uint16(headerRaw[12:14])
	finf.AlterCharIndex = binary.BigEndian.Uint16(headerRaw[14:16])
	finf.DefaultLeftWidth = headerRaw[16]
	finf.DefaultGlyphWidth = headerRaw[17]
	finf.DefaultCharWidth = headerRaw[18]
	finf.Encoding = headerRaw[19]
	finf.TGLPOffset = binary.BigEndian.Uint32(headerRaw[20:24])
	finf.CWDHOffset = binary.BigEndian.Uint32(headerRaw[24:28])
	finf.CMAPOffset = binary.BigEndian.Uint32(headerRaw[28:FINF_HEADER_SIZE])

	if Debug {
		pprint(finf)
		fmt.Printf("Read section total of %d bytes\n", headerEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header %d(inclusive) to %d(exclusive)\n", headerStart, headerEnd)
		fmt.Println()
	}
}

func (finf *FINF) Encode() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(finf.MagicHeader))
	binaryWrite(w, finf.SectionSize)
	binaryWrite(w, finf.FontType)
	binaryWrite(w, finf.Height)
	binaryWrite(w, finf.Width)
	binaryWrite(w, finf.Ascent)
	binaryWrite(w, finf.LineFeed)
	binaryWrite(w, finf.AlterCharIndex)
	binaryWrite(w, finf.DefaultLeftWidth)
	binaryWrite(w, finf.DefaultGlyphWidth)
	binaryWrite(w, finf.DefaultCharWidth)
	binaryWrite(w, finf.Encoding)
	binaryWrite(w, finf.TGLPOffset)
	binaryWrite(w, finf.CWDHOffset)
	binaryWrite(w, finf.CMAPOffset)
	w.Flush()

	assertEqual(FINF_HEADER_SIZE, len(buf.Bytes()))
	return buf.Bytes()
}
