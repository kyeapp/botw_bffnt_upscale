package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"

	"github.com/disintegration/imaging"
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

func (cfnt *CFNT) decode(raw []byte) {
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

func (finf *FINF_BFFNT) decode(raw []byte) {
	// Version 4 (BFFNT)
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

type TGLP_BFFNT struct { //    Offset  Size  Description
	MagicHeader      string // 0x00    0x04  Magic Header (TGLP)
	SectionSize      uint32 // 0x04    0x04  Section Size
	CellWidth        uint8  // 0x08    0x01  Cell Width
	CellHeight       uint8  // 0x09    0x01  Cell Height
	NumOfSheets      uint8  // 0x0A    0x01  Number of Sheets
	MaxCharWidth     uint8  // 0x0B    0x01  Max Character Width
	SheetSize        uint32 // 0x0C    0x04  Sheet Size
	BaselinePosition uint16 // 0x10    0x02  Baseline Position
	SheetImageFormat uint16 // 0x12    0x02  Sheet Image Format 0-13: (RGBA8, RGB8, RGBA5551, RGB565, RGBA4, LA8, HILO8, L8, A8, LA4, L4, A4, ETC1, ETC1A4)
	NumOfColumns     uint16 // 0x14    0x02  Number of Sheet columns
	NumOfRows        uint16 // 0x16    0x02  Number of Sheet rows
	SheetWidth       uint16 // 0x18    0x02  Sheet Width
	SheetHeight      uint16 // 0x1A    0x02  Sheet Height
	SheetDataOffset  uint32 // 0x1C    0x04  Sheet Data Offset
}

func (tglp *TGLP_BFFNT) decode(tglpRaw []byte, allRaw []byte) {
	raw := tglpRaw
	// Version 4 (BFFNT)
	tglp.MagicHeader = string(raw[0:4])
	tglp.SectionSize = binary.BigEndian.Uint32(raw[4:])
	tglp.CellWidth = raw[8] // byte == uint8
	tglp.CellHeight = raw[9]
	tglp.NumOfSheets = raw[10]
	tglp.MaxCharWidth = raw[11]
	tglp.SheetSize = binary.BigEndian.Uint32(raw[12:])
	tglp.BaselinePosition = binary.BigEndian.Uint16(raw[16:])
	tglp.SheetImageFormat = binary.BigEndian.Uint16(raw[18:])
	tglp.NumOfColumns = binary.BigEndian.Uint16(raw[20:])
	tglp.NumOfRows = binary.BigEndian.Uint16(raw[22:])
	tglp.SheetWidth = binary.BigEndian.Uint16(raw[24:])
	tglp.SheetHeight = binary.BigEndian.Uint16(raw[26:])
	tglp.SheetDataOffset = binary.BigEndian.Uint32(raw[28:])

	//DECODE IMAGE===========================================================
	start := tglp.SheetDataOffset
	end := tglp.SheetDataOffset + tglp.SheetSize
	data := allRaw[start:end]

	alphaImg := image.Alpha{
		Pix: data,
		// TODO: int conversion should end with a positive number
		Stride: int(tglp.SheetWidth),
		Rect:   image.Rect(0, 0, int(tglp.SheetWidth), int(tglp.SheetHeight)),
	}

	sheetW := int(tglp.wdtih)
	bytesPerPixel := 1 // TODO general pixel decoding for images that are encoded with 4 bytes PerPixel RGBA

	// Copied from KillzXGaming/Switch-Toolbox
	// Iterate through every pixel
	// Assumes only 1 byte per pixel (alpha only image)
	result := make([]bytes, int(tglp.SheetSize))
	swizzle := 1

	for y := 0; y < int(tglp.height); y++ {
		for x := 0; x < sheetW; x++ {
			pixelIndex := (y*sheetW + x) * bytesPerPixel
			swizzledPixelIndex := computeSwizzledPixelIndex()

			if swizzle {
				result[pixelIndex] = data[swizzledPixelIndex]
			} else {
				// deswizzle image
				result[swizzledPixelIndex] = data[pixelIndex]
			}
		}
	}

	// TODO rework this to use custom horizontal flip
	imgFlipped := imaging.FlipV(alphaImg)

	// convert back into an alpha pic
	buf := make([]uint8, tglp.SheetSize)
	for i := 0; i < tglp.SheetSize; i += 4 {
		alphaImg.Pix[i] = imgFlipped.Pix[i+3]
	}

	f, err := os.Create("outimage.png")
	handleErr(err)
	defer f.Close()

	// Encode to `PNG` with `DefaultCompression` level
	// then save to file
	err = png.Encode(f, alphaImg)
	handleErr(err)
	//==================================================================================

	fmt.Println("TGLP Header")
	pprint(tglp)
}

// computeSurfaceAddrFromCoordMacroTiled
func computeSwizzledPixelIndex(x uint, y uint, sample uint, pitch uint, height uint, tileMode AddrTileMode, pipSwizzle uint, backupSwizzle uint) uint {

	// var microTileThickness uint = computeSurfaceThickness(tileMode)
	var microTileThickness uint = 1

	var microTileBits uint = numSamples * bitsPerPixel * (microTileThickness * 64)
	var microTileBytes uint = (microTileBits + 7) / 8

	var pixelIndex uint = computePixelIndexWithinMicroTile(x, y, slice, bitsPerPixel, tileMode, IsDepth)
	var bytesPerSample uint = microTileBytes / numSamples
	var sampleOffset uint = 0
	var pixelOffset uint = 0
	var samplesPerSlice uint = 0
	var numSampleSplits uint = 0
	var sampleSlice uint = 0

	if hasDepth {
		sampleOffset = bitsPerPixel * sample
		pixelOffset = numSamples * bitsPerPixel * pixelIndex
	} else {
		sampleOffset = sample * (microTileBits / numSamples)
		pixelOffset = bitsPerPixel * pixelIndex
	}

	elemOffset := pixelOffset + sampleOffset

	if numSamples <= 1 || microTileBytes <= 2048 {
		samplesPerSlice = numSamples
		numSampleSplits = 1
		sampleSlice = 0
	} else {
		samplesPerSlice = 2048 / bytesPerSample
		numSampleSplits = numSamples / samplesPerSlice
		numSamples = samplesPerSlice

		var tileSliceBits uint = microTileBits / numSampleSplits
		sampleSlice = elemOffset / tileSliceBits
		elemOffset %= tileSliceBits
	}

	elemOffset = (elemOffset + 7) / 8

	var pipe uint = computePipeFromCoordWoRotation(x, y)
	var bank uint = computeBankFromCoordWoRotation(x, y)

	var swizzle_ uint = pipeSwizzle + 2*bankSwizzle
	var bankPipe uint = pipe + 2*bank
	var rotation uint = computeSurfaceRotationFromTileMode(tileMode)
	var sliceIn uint = slice

	if isThickMacroTiled(tileMode) != 0 {
		sliceIn >>= 2
	}

	bankPipe ^= 2*sampleSlice*3 ^ (swizzle_ + sliceIn*rotation)
	bankPipe %= 8

	pipe = bankPipe % 2
	bank = bankPipe / 2

	var sliceBytes uint = (height*pitch*microTileThickness*bitsPerPixel*numSamples + 7) / 8
	var sliceOffset uint = sliceBytes * (sampleSlice + numSampleSplits*slice) / microTileThickness

	var macroTilePitch uint = 32
	var macroTileHeight uint = 16

	switch tileMode {
	case AddrTileMode.ADDR_TM_2D_TILED_THIN2:
	case AddrTileMode.ADDR_TM_2B_TILED_THIN2:
		macroTilePitch = 16
		macroTileHeight = 32
		break
	case AddrTileMode.ADDR_TM_2D_TILED_THIN4:
	case AddrTileMode.ADDR_TM_2B_TILED_THIN4:
		macroTilePitch = 8
		macroTileHeight = 64
		break
	}

	var macroTilesPerRow uint = pitch / macroTilePitch
	var macroTileBytes uint = (numSamples*microTileThickness*bitsPerPixel*macroTileHeight*macroTilePitch + 7) / 8
	var macroTileIndexX uint = x / macroTilePitch
	var macroTileIndexY uint = y / macroTileHeight
	var macroTileOffset uint = (macroTileIndexX + macroTilesPerRow*macroTileIndexY) * macroTileBytes

	if isBankSwappedTileMode(tileMode) != 0 {
		var bankSwapWidth uint = computeSurfaceBankSwappedWidth(tileMode, bitsPerPixel, 1, pitch)
		var swapIndex uint = macroTilePitch * macroTileIndexX / bankSwapWidth
		bank ^= bankSwapOrder[swapIndex&3]
	}

	var totalOffset uint = elemOffset + ((macroTileOffset + sliceOffset) >> 3)
	res := bank<<9 || pipe<<8 || totalOffset&255 || (uint)((int)(totalOffset)&-256)<<3

	return res
}

type CWDH struct { //           Offset  Size                             Description
	MagicHeader string //       0x00    0x04                             Magic Header (CWDH)
	SectionSize uint32 //       0x04    0x04                             Section Size
	// StartIndex     uint16 // 0x08    0x02                             Start Index
	// EndIndex       uint16 // 0x0A    0x02                             End Index
	// NextCDWHOffset uint32 // 0x0C    0x04                             Next CWDH Offset
	// LeftWidth      uint8  // 0x10    3 * (EndIndex - StartIndex + 1)  Char Widths (3 bytes: Left, Glyph Width, Char Width)
	// GlyphWidth     uint8
	// CharWidth      uint8

	// First glyph included to keep things in line with the documentation
	glyphInfo
}

type glyphInfo struct {
	StartIndex     uint16
	EndIndex       uint16
	NextCWDHOffset uint32
	LeftWidth      uint8
	GlyphWidth     uint8
	CharWidth      uint8
}

func (cwdh *CWDH) decode(raw []byte) []glyphInfo {
	cwdh.MagicHeader = string(raw[0:4])
	cwdh.SectionSize = binary.BigEndian.Uint32(raw[4:])
	cwdh.StartIndex = binary.BigEndian.Uint16(raw[8:])
	cwdh.EndIndex = binary.BigEndian.Uint16(raw[10:])
	cwdh.NextCWDHOffset = binary.BigEndian.Uint32(raw[12:])
	cwdh.LeftWidth = raw[16]
	cwdh.GlyphWidth = raw[17]
	cwdh.CharWidth = raw[18]

	fmt.Println("CWDH Header")
	pprint(cwdh)
	return nil
}

type CMAP struct { //         Offset  Size  Description
	MagicHeader     string // 0x00    0x04  Magic Header (CMAP)
	SectionSize     uint32 // 0x04    0x04  Section Size
	CodeBegin       uint16 // 0x08    0x02  Code Begin
	CodeEnd         uint16 // 0x0A    0x02  Code End
	MappingMethod   uint16 // 0x0C    0x02  Mapping Method (0 = Direct, 1 = Table, 2 = Scan)
	UnknownReserved uint16 // 0x0E    0x02  Reserved?
	NextCMAPOffset  uint32 // 0x10    0x04  Next CMAP Offset
}

func (cmap *CMAP) decode(raw []byte) {
	cmap.MagicHeader = string(raw[0:4])
	cmap.SectionSize = binary.BigEndian.Uint32(raw[4:])
	cmap.CodeBegin = binary.BigEndian.Uint16(raw[8:])
	cmap.CodeEnd = binary.BigEndian.Uint16(raw[10:])
	cmap.MappingMethod = binary.BigEndian.Uint16(raw[12:])
	cmap.UnknownReserved = binary.BigEndian.Uint16(raw[14:])
	cmap.NextCMAPOffset = binary.BigEndian.Uint32(raw[16:])

	fmt.Println("CMAP Header")
	pprint(cmap)
}

// This BFFNT file is Breath of the Wild's NormalS_00.bffnt. The goal of the
// project is to create a bffnt encoder/decoder so I can upscale this font
const testBffntFile = "NormalS_00.bffnt"

func main() {
	rawBytes, err := ioutil.ReadFile(testBffntFile)
	handleErr(err)

	var cfnt CFNT
	cfnt.decode(rawBytes[0:20])

	var finf FINF_BFFNT
	finf.decode(rawBytes[20:52])

	var tglp TGLP_BFFNT
	tglp.decode(rawBytes[52:100], rawBytes)

	var cwdh CWDH
	// CWDHOffset skips the first 8 bytes that contain the CWDH Magic Header
	cwdh.decode(rawBytes[finf.CWDHOffset-8:])

	var cmap CMAP
	// CMAPOffset skips the first 8 bytes that contain the CWDH Magic Header
	cmap.decode(rawBytes[finf.CMAPOffset-8:])
}
