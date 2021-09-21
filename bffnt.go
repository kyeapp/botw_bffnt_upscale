package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Resources
// https://www.3dbrew.org/wiki/BCFNT#Version_4_.28BFFNT.29

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func pprint(s interface{}) {
	jsonBytes, err := json.MarshalIndent(s, "", "  ")
	handleErr(err)

	fmt.Printf("%s\n", string(jsonBytes))
}

type CFNT struct { //       Offset  Size  Description
	MagicHeader   string // 0x00    0x04  Magic Header (either CFNT or CFNU or FFNT)
	Endianness    string // 0x04    0x02  Endianness (0xFEFF = little, 0xFFFE = big)
	SectionSize   uint16 // 0x06    0x02  Header Size
	Version       string // 0x08    0x04  Version (observed to be 0x03000000)
	TotalFileSize uint32 // 0x0C    0x04  File size (the total)
	BlockReadNum  uint32 // 0x10    0x04  Number of "blocks" to read
}

func (cfnt CFNT) decode(raw []byte) {
	cfnt.MagicHeader = string(raw[0:4])
	cfnt.Endianness = fmt.Sprintf("%X", binary.BigEndian.Uint16(raw[4:]))
	cfnt.SectionSize = binary.BigEndian.Uint16(raw[6:])
	cfnt.Version = fmt.Sprintf("%X", binary.BigEndian.Uint32(raw[8:]))
	cfnt.TotalFileSize = binary.BigEndian.Uint32(raw[12:])
	cfnt.BlockReadNum = binary.BigEndian.Uint32(raw[16:])

	if cfnt.Endianness != "FEFF" {
		panic("only little endian is supported")
	}

	fmt.Println("CFNT Header")
	pprint(cfnt)
}

type FINF_BFFNT struct { //  Offset  Size  Description
	MagicHeader    string // 0x00    0x04  Magic Header (FINF)
	SectionSize    uint32 // 0x04    0x04  Section Size
	FontType       uint8  // 0x08    0x01  Font Type
	Height         uint8  // 0x09    0x01  Height
	Width          uint8  // 0x0A    0x01  Width
	Ascent         uint8  // 0x0B    0x01  Ascent
	LineFeed       uint16 // 0x0C    0x02  Line Feed
	AlterCharIndex uint16 // 0x0E    0x02  Alter Char Index
	LeftWidth      uint8  // 0x10    0x03  Default Width (3 bytes: Left, Glyph Width, Char Width)
	GlyphWidth     uint8
	CharWidth      uint8
	Encoding       uint8  // 0x13    0x01  Encoding
	TGLPOffset     uint32 // 0x14    0x04  TGLP Offset
	CWDHOffset     uint32 // 0x18    0x04  CWDH Offset
	CMAPOffset     uint32 // 0x1C    0x04  CMAP Offset
}

func (finf FINF_BFFNT) decode(raw []byte) {
	finf.MagicHeader = string(raw[0:4])
	finf.SectionSize = binary.BigEndian.Uint32(raw[4:])
	finf.FontType = raw[8] // byte == uint8
	finf.Height = raw[9]
	finf.Width = raw[10]
	finf.Ascent = raw[11]
	finf.LineFeed = binary.BigEndian.Uint16(raw[12:])
	finf.AlterCharIndex = binary.BigEndian.Uint16(raw[14:])
	finf.LeftWidth = raw[16]
	finf.GlyphWidth = raw[17]
	finf.CharWidth = raw[18]
	finf.Encoding = raw[19]
	finf.TGLPOffset = binary.BigEndian.Uint32(raw[20:])
	finf.CWDHOffset = binary.BigEndian.Uint32(raw[24:])
	finf.CMAPOffset = binary.BigEndian.Uint32(raw[28:])

	fmt.Println("FINF Header")
	pprint(finf)
}

// This BFFNT file is Breath of the Wild's NormalS_00.bffnt. The goal of the
// project is to create a bffnt encoder/decoder so I can upscale this font
const testBffntFile = "NormalS_00.bffnt"

func main() {
	rawBytes, err := ioutil.ReadFile(testBffntFile)
	handleErr(err)

	var cfnt CFNT
	cfnt.decode(rawBytes[0:20])

	fmt.Println()

	var finf FINF_BFFNT
	finf.decode(rawBytes[20:52])
}

// WIP

// Offset  Size  Description
// 0x00    0x04  Magic Header (TGLP)
// 0x04    0x04  Section Size
// 0x08    0x01  Cell Width
// 0x09    0x01  Cell Height
// 0x0A    0x01  Number of Sheets
// 0x0B    0x01  Max Character Width
// 0x0C    0x04  Sheet Size
// 0x10    0x02  Baseline Position
// 0x12    0x02  Sheet Image Format 0-13: (RGBA8, RGB8, RGBA5551, RGB565, RGBA4, LA8, HILO8, L8, A8, LA4, L4, A4, ETC1, ETC1A4)
// 0x14    0x02  Number of Sheet columns
// 0x16    0x02  Number of Sheet rows
// 0x18    0x02  Sheet Width
// 0x1A    0x02  Sheet Height
// 0x1C    0x04  Sheet Data Offset
type TGLP_BFFNT struct {
	MagicHeader      string
	SectionSize      uint32
	CellWidth        uint8
	CellHeight       uint8
	NumOfSheets      uint8
	MaxCharWidth     uint8
	SheetSize        uint32
	BaselinePosition uint16
	SheetImageFormat uint16
	NumOfColumns     uint16
	NumOfRows        uint16
	SheetWidth       uint16
	SheetHeight      uint16
	SheetDataOffset  uint32
}

func (t TGLP_BFFNT) decode(raw []byte) {
	// temporary variables
	a := raw[0:4]
	// b := binary.BigEndian.Uint32(raw[4:])
	// c := binary.BigEndian.Uint8(raw[6:])
	// d := binary.BigEndian.Uint8(raw[6:])
	// e := binary.BigEndian.Uint8(raw[6:])
	// f := binary.BigEndian.Uint8(raw[6:])
	// g := binary.BigEndian.Uint32(raw[8:])
	// h := binary.BigEndian.Uint32(raw[12:])
	// i := binary.BigEndian.Uint16(raw[16:])
	// j := binary.BigEndian.Uint16(raw[16:])
	// k := binary.BigEndian.Uint16(raw[16:])
	// l := binary.BigEndian.Uint16(raw[16:])
	// m := binary.BigEndian.Uint16(raw[16:])
	// n := binary.BigEndian.Uint16(raw[16:])
	// o := binary.BigEndian.Uint32(raw[16:])

	t.MagicHeader = string(a)

	pprint(t)
}
