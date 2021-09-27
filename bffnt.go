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

type AddrTileMode uint

const (
	ADDR_TM_LINEAR_GENERAL AddrTileMode = iota
	ADDR_TM_LINEAR_ALIGNED
	ADDR_TM_1D_TILED_THIN1
	ADDR_TM_1D_TILED_THICK
	ADDR_TM_2D_TILED_THIN1
	ADDR_TM_2D_TILED_THIN2
	ADDR_TM_2D_TILED_THIN4
	ADDR_TM_2D_TILED_THICK
	ADDR_TM_2B_TILED_THIN1
	ADDR_TM_2B_TILED_THIN2
	ADDR_TM_2B_TILED_THIN4
	ADDR_TM_2B_TILED_THICK
	ADDR_TM_3D_TILED_THIN1
	ADDR_TM_3D_TILED_THICK
	ADDR_TM_3B_TILED_THIN1
	ADDR_TM_3B_TILED_THICK
	ADDR_TM_2D_TILED_XTHICK
	ADDR_TM_3D_TILED_XTHICK
	ADDR_TM_POWER_SAVE
	ADDR_TM_COUNT
)

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
		panic("big endian not supported")
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

	fmt.Println("TGLP Header")
	pprint(tglp)

	//DECODE IMAGE===========================================================
	start := tglp.SheetDataOffset
	end := tglp.SheetDataOffset + tglp.SheetSize
	data := allRaw[start:end]

	depth := uint(1)
	sw := uint(tglp.SheetWidth)
	sh := uint(tglp.SheetHeight)
	format_ := uint(1)
	aa := uint(0)
	use := uint(2)
	tileMode := uint(4)
	swizzle_ := uint(0)
	bpp := uint(8)
	slice := uint(0)
	sample := uint(0)
	deswizzledImage := deswizzle(sw, sh, depth, sh, format_, aa, use, tileMode, swizzle_, sw, bpp, slice, sample, data)

	// // TODO rework this to use custom horizontal flip
	alphaImg := image.Alpha{
		Pix: deswizzledImage,
		// TODO: int conversion should end with a positive number
		Stride: int(tglp.SheetWidth),
		Rect:   image.Rect(0, 0, int(tglp.SheetWidth), int(tglp.SheetHeight)),
	}

	imgFlipped := imaging.FlipV(alphaImg.SubImage(alphaImg.Rect))

	// convert back into alphaImg
	for i := uint32(0); i < tglp.SheetSize; i++ {
		alphaImg.Pix[i] = imgFlipped.Pix[4*i+3]
	}

	f, err := os.Create("outimage.png")
	handleErr(err)
	defer f.Close()

	// Encode to `PNG` with `DefaultCompression` level
	// then save to file
	err = png.Encode(f, alphaImg.SubImage(alphaImg.Rect))
	handleErr(err)
	//==================================================================================
}

func deswizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, data []byte) []byte {
	return swizzleSurface(width, height, depth, format, aa, use, tileMode, swizzle_, pitch, bpp, slice, sample, data, false)
}
func swizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, byte, data []byte) []byte {
	return swizzleSurface(width, height, depth, format, aa, use, tileMode, swizzle_, pitch, bpp, slice, sample, data, true)
}

// Copied from KillzXGaming/Switch-Toolbox
func swizzleSurface(width uint, height uint, depth uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, data []byte, swizzle bool) []byte {
	var bytesPerPixel uint = bpp / 8
	result := make([]byte, len(data))

	// uint pipeSwizzle, bankSwizzle, pos_;
	// ulong pos;

	// if (IsFormatBCN((GX2SurfaceFormat)format))
	// {
	//     width = (width + 3) / 4;
	//     height = (height + 3) / 4;
	// }

	// pipeSwizzle = (swizzle_ >> 8) & 1;
	// bankSwizzle = (swizzle_ >> 9) & 3;

	// if (depth > 1)
	// {
	// //     bankSwizzle = (uint)(slice % 4);
	// }

	// tileMode = GX2TileModeToAddrTileMode(tileMode);

	// var IsDepth bool = (use & 4) != 0
	isDepth := false
	// var numSamples uint = (uint)(1 << (int)(aa))

	var swizzledPixelIndex uint

	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			// if tileMode == 0 || tileMode == 1 {
			// 	pos = computeSurfaceAddrFromCoordLinear((uint)x, (uint)y, slice, sample, bytesPerPixel, pitch, height, depth);
			// 	panic("unsupported tile mode")
			// } else if tileMode == 2 || tileMode == 3 {
			// 	pos = computeSurfaceAddrFromCoordMicroTiled((uint)x, (uint)y, slice, bpp, pitch, height, (AddrTileMode)tileMode, IsDepth);
			// 	panic("unsupported tile mode")
			// } else {
			// 	pos = computeSurfaceAddrFromCoordMacroTiled((uint)x, (uint)y, slice, sample, bpp, pitch, height, numSamples, (AddrTileMode)tileMode, IsDepth, pipeSwizzle, bankSwizzle);
			swizzledPixelIndex = computeSwizzledPixelIndex(x, y, bpp, pitch, height, ADDR_TM_2D_TILED_THIN1, isDepth)
			// }

			var pixelIndex uint = (y*width + x) * bytesPerPixel
			dataLen := (uint)(len(data))
			if pixelIndex+bytesPerPixel <= dataLen && swizzledPixelIndex+bytesPerPixel <= dataLen {
				if swizzle {
					// swizzle
					result[swizzledPixelIndex] = data[pixelIndex]
				} else {
					// deswizzle
					result[pixelIndex] = data[swizzledPixelIndex]
				}
			}
		}
	}

	return result
}

// computeSurfaceAddrFromCoordMacroTiled(uint x, uint y, uint slice, uint
// sample, uint bpp, uint pitch, uint height, uint numSamples, AddrTileMode
// tileMode, bool IsDepth, uint pipeSwizzle, uint bankSwizzle)
func computeSwizzledPixelIndex(x uint, y uint, bpp uint, pitch uint, height uint, tileMode AddrTileMode, isDepth bool) uint {
	var pipeSwizzle uint = 0
	var bankSwizzle uint = 0
	var numSamples uint = 1
	var sample uint = 0
	var slice uint = 0
	var microTileThickness uint = computeSurfaceThickness(tileMode)

	var microTileBits uint = numSamples * bpp * (microTileThickness * 64)
	var microTileBytes uint = (microTileBits + 7) / 8

	var pixelIndex uint = computePixelIndexWithinMicroTile(x, y, slice, bpp, tileMode, isDepth)
	var bytesPerSample uint = microTileBytes / numSamples
	var sampleOffset uint = 0
	var pixelOffset uint = 0
	var samplesPerSlice uint = 0
	var numSampleSplits uint = 0
	var sampleSlice uint = 0

	// if hasDepth {
	// 	sampleOffset = bpp * sample
	// 	pixelOffset = numSamples * bpp * pixelIndex
	// } else {
	sampleOffset = sample * (microTileBits / numSamples)
	pixelOffset = bpp * pixelIndex
	// }

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

	var sliceBytes uint = (height*pitch*microTileThickness*bpp*numSamples + 7) / 8
	var sliceOffset uint = sliceBytes * (sampleSlice + numSampleSplits*slice) / microTileThickness

	macroTilePitch, macroTileHeight := computeMacroPitchAndHeight(tileMode)

	var macroTilesPerRow uint = pitch / macroTilePitch
	var macroTileBytes uint = (numSamples*microTileThickness*bpp*macroTileHeight*macroTilePitch + 7) / 8
	var macroTileIndexX uint = x / macroTilePitch
	var macroTileIndexY uint = y / macroTileHeight
	var macroTileOffset uint = (macroTileIndexX + macroTilesPerRow*macroTileIndexY) * macroTileBytes

	// if isBankSwappedTileMode(tileMode) != 0 {
	// 	var bankSwapWidth uint = computeSurfaceBankSwappedWidth(tileMode, bpp, 1, pitch)
	// 	var swapIndex uint = macroTilePitch * macroTileIndexX / bankSwapWidth
	// 	bank ^= bankSwapOrder[swapIndex&3]
	// }

	var totalOffset uint = elemOffset + ((macroTileOffset + sliceOffset) >> 3)
	res := bank<<9 | pipe<<8 | totalOffset&255 | (uint)((int)(totalOffset)&-256)<<3

	return res
}

func computePixelIndexWithinMicroTile(x uint, y uint, z uint, bpp uint, tileMode AddrTileMode, isDepth bool) uint {
	var pixelBit0 uint = 0
	var pixelBit1 uint = 0
	var pixelBit2 uint = 0
	var pixelBit3 uint = 0
	var pixelBit4 uint = 0
	var pixelBit5 uint = 0
	var pixelBit6 uint = 0
	var pixelBit7 uint = 0
	var pixelBit8 uint = 0
	var thickness uint = computeSurfaceThickness(tileMode)

	if isDepth {
		pixelBit0 = x & 1
		pixelBit1 = y & 1
		pixelBit2 = (x & 2) >> 1
		pixelBit3 = (y & 2) >> 1
		pixelBit4 = (x & 4) >> 2
		pixelBit5 = (y & 4) >> 2
	} else {
		switch bpp {
		case 8:
			pixelBit0 = x & 1
			pixelBit1 = (x & 2) >> 1
			pixelBit2 = (x & 4) >> 2
			pixelBit3 = (y & 2) >> 1
			pixelBit4 = y & 1
			pixelBit5 = (y & 4) >> 2
			break
		case 0x10:
			pixelBit0 = x & 1
			pixelBit1 = (x & 2) >> 1
			pixelBit2 = (x & 4) >> 2
			pixelBit3 = y & 1
			pixelBit4 = (y & 2) >> 1
			pixelBit5 = (y & 4) >> 2
			break
		case 0x20:
			fallthrough
		case 0x60:
			pixelBit0 = x & 1
			pixelBit1 = (x & 2) >> 1
			pixelBit2 = y & 1
			pixelBit3 = (x & 4) >> 2
			pixelBit4 = (y & 2) >> 1
			pixelBit5 = (y & 4) >> 2
			break
		case 0x40:
			pixelBit0 = x & 1
			pixelBit1 = y & 1
			pixelBit2 = (x & 2) >> 1
			pixelBit3 = (x & 4) >> 2
			pixelBit4 = (y & 2) >> 1
			pixelBit5 = (y & 4) >> 2
			break
		case 0x80:
			pixelBit0 = y & 1
			pixelBit1 = x & 1
			pixelBit2 = (x & 2) >> 1
			pixelBit3 = (x & 4) >> 2
			pixelBit4 = (y & 2) >> 1
			pixelBit5 = (y & 4) >> 2
			break
		default:
			pixelBit0 = x & 1
			pixelBit1 = (x & 2) >> 1
			pixelBit2 = y & 1
			pixelBit3 = (x & 4) >> 2
			pixelBit4 = (y & 2) >> 1
			pixelBit5 = (y & 4) >> 2
			break
		}
	}

	if thickness > 1 {
		pixelBit6 = z & 1
		pixelBit7 = (z & 2) >> 1
	}

	if thickness == 8 {
		pixelBit8 = (z & 4) >> 2
	}

	return (pixelBit8 << 8) | (pixelBit7 << 7) | (pixelBit6 << 6) | 32*pixelBit5 | 16*pixelBit4 | 8*pixelBit3 | 4*pixelBit2 | pixelBit0 | 2*pixelBit1
}

func computeSurfaceThickness(tileMode AddrTileMode) uint {
	switch tileMode {
	case ADDR_TM_1D_TILED_THICK:
		fallthrough
	case ADDR_TM_2D_TILED_THICK:
		fallthrough
	case ADDR_TM_2B_TILED_THICK:
		fallthrough
	case ADDR_TM_3D_TILED_THICK:
		fallthrough
	case ADDR_TM_3B_TILED_THICK:
		return 4
	case ADDR_TM_2D_TILED_XTHICK:
		fallthrough
	case ADDR_TM_3D_TILED_XTHICK:
		return 8
	default:
		return 1
	}
}

func computePipeFromCoordWoRotation(x uint, y uint) uint {
	return ((y >> 3) ^ (x >> 3)) & 1
}

func computeBankFromCoordWoRotation(x uint, y uint) uint {
	return ((y>>5)^(x>>3))&1 | 2*(((y>>4)^(x>>4))&1)
}

func computeSurfaceRotationFromTileMode(tileMode AddrTileMode) uint {
	switch tileMode {
	case ADDR_TM_2D_TILED_THIN1:
		fallthrough
	case ADDR_TM_2D_TILED_THIN2:
		fallthrough
	case ADDR_TM_2D_TILED_THIN4:
		fallthrough
	case ADDR_TM_2D_TILED_THICK:
		fallthrough
	case ADDR_TM_2B_TILED_THIN1:
		fallthrough
	case ADDR_TM_2B_TILED_THIN2:
		fallthrough
	case ADDR_TM_2B_TILED_THIN4:
		fallthrough
	case ADDR_TM_2B_TILED_THICK:
		return 2
	case ADDR_TM_3D_TILED_THIN1:
		fallthrough
	case ADDR_TM_3D_TILED_THICK:
		fallthrough
	case ADDR_TM_3B_TILED_THIN1:
		fallthrough
	case ADDR_TM_3B_TILED_THICK:
		return 1
	default:
		return 0
	}
}

func isThickMacroTiled(tileMode AddrTileMode) uint {
	switch tileMode {
	case ADDR_TM_2D_TILED_THICK:
		fallthrough
	case ADDR_TM_2B_TILED_THICK:
		fallthrough
	case ADDR_TM_3D_TILED_THICK:
		fallthrough
	case ADDR_TM_3B_TILED_THICK:
		return 1
	default:
		return 0
	}
}

func computeMacroPitchAndHeight(tileMode AddrTileMode) (pitch uint, height uint) {
	var macroTilePitch uint = 32
	var macroTileHeight uint = 16

	switch tileMode {
	case ADDR_TM_2D_TILED_THIN2:
		fallthrough
	case ADDR_TM_2B_TILED_THIN2:
		macroTilePitch = 16
		macroTileHeight = 32
		break
	case ADDR_TM_2D_TILED_THIN4:
		fallthrough
	case ADDR_TM_2B_TILED_THIN4:
		macroTilePitch = 8
		macroTileHeight = 64
		break
	}

	return macroTilePitch, macroTileHeight
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

const (
	Direct uint16 = 0
	Table  uint16 = 1
	Scan   uint16 = 2
)

func (cmap *CMAP) decode(allRaw []byte, offset uint32) []CMAP {
	raw := allRaw[offset:]
	cmap.MagicHeader = string(raw[0:4])
	cmap.SectionSize = binary.BigEndian.Uint32(raw[4:])
	cmap.CodeBegin = binary.BigEndian.Uint16(raw[8:])
	cmap.CodeEnd = binary.BigEndian.Uint16(raw[10:])
	// TODO: put mapping into its own type. it would be easier to read case statements.
	cmap.MappingMethod = binary.BigEndian.Uint16(raw[12:])
	cmap.UnknownReserved = binary.BigEndian.Uint16(raw[14:])
	cmap.NextCMAPOffset = binary.BigEndian.Uint32(raw[16:])

	fmt.Println("CMAP Header")
	pprint(cmap)

	// TODO sectionsize verification

	// direct mapping is used if all the characters in the range are used. The
	// reserved data is characterOffset. Character offset is needed if the
	// direct map is not the first map to be read. Instead of storing an array
	// of the character's index, we can save (CodeEnd - CodeStart + 1) uint16s
	// worth of bytes by just storing an offset and calculating the index. With
	// each new character in a direct character map, the character's index is
	// incremented by 1. The character offset should be equal to the total
	// number of characters read in from the previous CMAPs.
	switch cmap.MappingMethod {
	case 0: //direct mapping
		characterOffset := cmap.UnknownReserved
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charIdx := i - cmap.CodeBegin + characterOffset
			fmt.Printf("direct %c %d\n", rune(i), charIdx)
		}
		break

	// table mapping is used when there are unused characters in the range of
	// characters the next (CodeEnd - CodeStart + 1) amount of bytes. An arrray
	// of index that starts after the cmap header is included. Unused
	// characters will have an index of MaxUint16 (65535).
	case 1: //table maping
		cmapIndex := 20
		for i := cmap.CodeBegin; i <= cmap.CodeEnd; i++ {
			charIdx := binary.BigEndian.Uint16(raw[cmapIndex:])
			if charIdx != 65535 { // math.MaxUint16
				fmt.Printf("table %#U %d\n", rune(i), charIdx)
			}
			cmapIndex += 2
		}
		break

	case 2: //scan
		break
	default:
		panic("unknown mapping method")
	}

	if cmap.NextCMAPOffset == 0 {
		return nil
	}

	cmap.decode(allRaw, cmap.NextCMAPOffset-8)

	return nil
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
	tglp.decode(rawBytes[52:84], rawBytes)

	var cmap CMAP
	// // CMAPOffset skips the first 8 bytes that contain the CMAP Magic Header
	cmap.decode(rawBytes, finf.CMAPOffset-8)

	// var cwdh CWDH
	// // CWDHOffset skips the first 8 bytes that contain the CWDH Magic Header
	// cwdh.decode(rawBytes[finf.CWDHOffset-8:])

}
