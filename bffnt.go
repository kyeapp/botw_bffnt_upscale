package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io/ioutil"

	"github.com/disintegration/imaging"
)

var debug bool

const (
	// number of bytes for each header size
	CFNT_HEADER_SIZE = 20
	FINF_HEADER_SIZE = 32
	TGLP_HEADER_SIZE = 32
	CWDH_HEADER_SIZE = 16
	CMAP_HEADER_SIZE = 20
)

// Resources
// https://www.3dbrew.org/wiki/BCFNT#Version_4_.28BFFNT.29
// http://wiki.tockdom.com/wiki/BRFNT_(File_Format)
// https://github.com/KillzXGaming/Switch-Toolbox/blob/12dfbaadafb1ebcd2e07d239361039a8d05df3f7/File_Format_Library/FileFormats/Font/BXFNT/FontKerningTable.cs

func assertEqual(expected int, actual int) {
	if expected != actual {
		panic(fmt.Errorf("%d(actual) does not equal %d(expected)\n", actual, expected))
	}
}

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
	Endianness    uint16 // 0x04    0x02  Endianness (0xFEFF = little, 0xFFFE = big)
	SectionSize   uint16 // 0x06    0x02  Header Size
	Version       uint32 // 0x08    0x04  Version (observed to be 0x03000000)
	TotalFileSize uint32 // 0x0C    0x04  File size (the total)
	BlockReadNum  uint32 // 0x10    0x04  Number of "blocks" to read

	// It looks like BlockReadNum is always some multiple of 2^16 (65536 in
	// decimal. 0x10000 in HEX). Unclear wether this can break a font. It might
	// be that its a suggestion to the system to it can block read at a time.
	// perhaps it is ok to change this number around. Change this bit and see if botw crashes.

	// remainder := (cfnt.TotalFileSize % 65536)
	// quotient := (cfnt.TotalFileSize - remainder) / 65536
	// calculatedBlockReadNum := int((quotient + 1) * 65536)
	// fmt.Println(tglp.SheetSize)
	// fmt.Println(remainder)
	// fmt.Println(quotient)
	// fmt.Println(calculatedBlockReadNum)
	// assertEqual(calculatedBlockReadNum, int(cfnt.BlockReadNum))
}

func (cfnt *CFNT) decode(raw []byte) {
	headerStart := 0
	headerEnd := headerStart + CFNT_HEADER_SIZE
	headerRaw := raw[headerStart:headerEnd]
	assertEqual(CFNT_HEADER_SIZE, len(headerRaw))

	cfnt.MagicHeader = string(headerRaw[0:4])
	cfnt.Endianness = binary.BigEndian.Uint16(headerRaw[4:6])
	cfnt.SectionSize = binary.BigEndian.Uint16(headerRaw[6:8])
	cfnt.Version = binary.BigEndian.Uint32(headerRaw[8:12])
	cfnt.TotalFileSize = binary.BigEndian.Uint32(headerRaw[12:16])
	cfnt.BlockReadNum = binary.BigEndian.Uint32(headerRaw[16:CFNT_HEADER_SIZE])

	if debug {
		pprint(cfnt)
		fmt.Printf("Read section total of %d bytes\n", headerEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header %d(inclusive) to %d(exclusive)\n", headerStart, headerEnd)
		fmt.Println()
	}
}

func (cfnt *CFNT) encode() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(cfnt.MagicHeader))
	_ = binary.Write(w, binary.BigEndian, cfnt.Endianness)
	_ = binary.Write(w, binary.BigEndian, cfnt.SectionSize)
	_ = binary.Write(w, binary.BigEndian, cfnt.Version)
	_ = binary.Write(w, binary.BigEndian, cfnt.TotalFileSize)
	_ = binary.Write(w, binary.BigEndian, cfnt.BlockReadNum)
	w.Flush()

	assertEqual(CFNT_HEADER_SIZE, len(buf.Bytes()))
	return buf.Bytes()
}

type FINF_BFFNT struct { //  Offset  Size  Description
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
func (finf *FINF_BFFNT) decode(raw []byte) {
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

	if debug {
		pprint(finf)
		fmt.Printf("Read section total of %d bytes\n", headerEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header %d(inclusive) to %d(exclusive)\n", headerStart, headerEnd)
		fmt.Println()
	}
}

func (finf *FINF_BFFNT) encode() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(finf.MagicHeader))
	_ = binary.Write(w, binary.BigEndian, finf.SectionSize)
	_ = binary.Write(w, binary.BigEndian, finf.FontType)
	_ = binary.Write(w, binary.BigEndian, finf.Height)
	_ = binary.Write(w, binary.BigEndian, finf.Width)
	_ = binary.Write(w, binary.BigEndian, finf.Ascent)
	_ = binary.Write(w, binary.BigEndian, finf.LineFeed)
	_ = binary.Write(w, binary.BigEndian, finf.AlterCharIndex)
	_ = binary.Write(w, binary.BigEndian, finf.DefaultLeftWidth)
	_ = binary.Write(w, binary.BigEndian, finf.DefaultGlyphWidth)
	_ = binary.Write(w, binary.BigEndian, finf.DefaultCharWidth)
	_ = binary.Write(w, binary.BigEndian, finf.Encoding)
	_ = binary.Write(w, binary.BigEndian, finf.TGLPOffset)
	_ = binary.Write(w, binary.BigEndian, finf.CWDHOffset)
	_ = binary.Write(w, binary.BigEndian, finf.CMAPOffset)
	w.Flush()

	assertEqual(FINF_HEADER_SIZE, len(buf.Bytes()))
	return buf.Bytes()
}

type TGLP_BFFNT struct { //    Offset  Size  Description
	MagicHeader      string        // 0x00    0x04  Magic Header (TGLP)
	SectionSize      uint32        // 0x04    0x04  Section Size
	CellWidth        uint8         // 0x08    0x01  Cell Width
	CellHeight       uint8         // 0x09    0x01  Cell Height
	NumOfSheets      uint8         // 0x0A    0x01  Number of Sheets
	MaxCharWidth     uint8         // 0x0B    0x01  Max Character Width
	SheetSize        uint32        // 0x0C    0x04  Sheet Size
	BaselinePosition uint16        // 0x10    0x02  Baseline Position
	SheetImageFormat uint16        // 0x12    0x02  Sheet Image Format 0-13: (RGBA8, RGB8, RGBA5551, RGB565, RGBA4, LA8, HILO8, L8, A8, LA4, L4, A4, ETC1, ETC1A4)
	NumOfColumns     uint16        // 0x14    0x02  Number of Sheet columns
	NumOfRows        uint16        // 0x16    0x02  Number of Sheet rows
	SheetWidth       uint16        // 0x18    0x02  Sheet Width
	SheetHeight      uint16        // 0x1A    0x02  Sheet Height
	SheetDataOffset  uint32        // 0x1C    0x04  Sheet Data Offset
	AllSheetData     []byte        // raw bytes of all data sheets
	SheetData        []image.Alpha // separated unswizzled images
}

// Version 4 (BFFNT)
// The input for TGLP decode is the entire BFFNT file in the form of a byte
// array ([]byte).
func (tglp *TGLP_BFFNT) decode(raw []byte) {
	headerStart := CFNT_HEADER_SIZE + FINF_HEADER_SIZE
	headerEnd := headerStart + TGLP_HEADER_SIZE
	headerRaw := raw[headerStart:headerEnd]
	assertEqual(TGLP_HEADER_SIZE, len(headerRaw))
	tglp.decodeHeader(headerRaw)

	totalSheetDataSize := int(tglp.SheetSize) * int(tglp.NumOfSheets)
	dataStart := int(tglp.SheetDataOffset)
	dataEnd := dataStart + totalSheetDataSize
	tglp.AllSheetData = raw[dataStart:dataEnd]

	// NOT TO SCALE representation of a portion of the bffnt file in raw bytes
	// for visual purposes
	//                      |-------------TGLP section size---------------------------|
	// CFNT   FINF          TGLP header    padding              tglp SheetDataOffset
	// |      |             |              |                    |
	// aaaaaa bbbbbbbbbbbbb cccccccccccccc 00000000000000000000 ddddddddddddddddddddddd
	padding := int(tglp.SheetDataOffset) - CFNT_HEADER_SIZE - FINF_HEADER_SIZE - TGLP_HEADER_SIZE
	calculatedTGLPSectionSize := TGLP_HEADER_SIZE + padding + len(tglp.AllSheetData)
	assertEqual(int(tglp.SectionSize), calculatedTGLPSectionSize)

	tglp.decodeSheets()
	if debug {
		fmt.Printf("Read section total of %d bytes\n", dataEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header      %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("padding     %-8d to  %d\n", headerEnd, dataStart)
		fmt.Printf("image data  %-8d to  %d\n", dataStart, dataEnd)
		fmt.Println()
	}
}

func (tglp *TGLP_BFFNT) decodeHeader(raw []byte) {
	tglp.MagicHeader = string(raw[0:4])
	tglp.SectionSize = binary.BigEndian.Uint32(raw[4:8])
	tglp.CellWidth = raw[8] // byte == uint8
	tglp.CellHeight = raw[9]
	tglp.NumOfSheets = raw[10]
	tglp.MaxCharWidth = raw[11]
	tglp.SheetSize = binary.BigEndian.Uint32(raw[12:16])
	tglp.BaselinePosition = binary.BigEndian.Uint16(raw[16:18])
	tglp.SheetImageFormat = binary.BigEndian.Uint16(raw[18:20])
	tglp.NumOfColumns = binary.BigEndian.Uint16(raw[20:22])
	tglp.NumOfRows = binary.BigEndian.Uint16(raw[22:24])
	tglp.SheetWidth = binary.BigEndian.Uint16(raw[24:26])
	tglp.SheetHeight = binary.BigEndian.Uint16(raw[26:28])
	tglp.SheetDataOffset = binary.BigEndian.Uint32(raw[28:TGLP_HEADER_SIZE])

	if debug {
		pprint(tglp)
	}
}

// TODO: decode multiple sheets
func (tglp *TGLP_BFFNT) decodeSheets() {
	totalSheetBytes := int(tglp.NumOfSheets) * int(tglp.SheetSize)
	assertEqual(totalSheetBytes, len(tglp.AllSheetData))

	sheetData := tglp.AllSheetData
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
	deswizzledImage := deswizzle(sw, sh, depth, sh, format_, aa, use, tileMode, swizzle_, sw, bpp, slice, sample, sheetData)

	// TODO rework this to use custom horizontal flip
	alphaImg := image.Alpha{
		Pix:    deswizzledImage,
		Stride: int(tglp.SheetWidth),
		Rect:   image.Rect(0, 0, int(tglp.SheetWidth), int(tglp.SheetHeight)),
	}

	// imaging.FlipV returns an NRGBA image
	imgFlipped := imaging.FlipV(alphaImg.SubImage(alphaImg.Rect))

	// convert back into alphaImg
	for i := uint32(0); i < tglp.SheetSize; i++ {
		alphaImg.Pix[i] = imgFlipped.Pix[4*i+3]
	}

	tglp.SheetData = append(tglp.SheetData, alphaImg)

	// f, err := os.Create("outimage.png")
	// handleErr(err)
	// defer f.Close()

	// // Encode to `PNG` with `DefaultCompression` level
	// // then save to file
	// err = png.Encode(f, alphaImg.SubImage(alphaImg.Rect))
	// handleErr(err)

}

func (tglp *TGLP_BFFNT) encodeHeader() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(tglp.MagicHeader))
	_ = binary.Write(w, binary.BigEndian, tglp.SectionSize)
	_ = binary.Write(w, binary.BigEndian, tglp.CellWidth)
	_ = binary.Write(w, binary.BigEndian, tglp.CellHeight)
	_ = binary.Write(w, binary.BigEndian, tglp.NumOfSheets)
	_ = binary.Write(w, binary.BigEndian, tglp.MaxCharWidth)
	_ = binary.Write(w, binary.BigEndian, tglp.SheetSize)
	_ = binary.Write(w, binary.BigEndian, tglp.BaselinePosition)
	_ = binary.Write(w, binary.BigEndian, tglp.SheetImageFormat)
	_ = binary.Write(w, binary.BigEndian, tglp.NumOfColumns)
	_ = binary.Write(w, binary.BigEndian, tglp.NumOfRows)
	_ = binary.Write(w, binary.BigEndian, tglp.SheetWidth)
	_ = binary.Write(w, binary.BigEndian, tglp.SheetHeight)
	_ = binary.Write(w, binary.BigEndian, tglp.SheetDataOffset)
	w.Flush()

	assertEqual(TGLP_HEADER_SIZE, len(buf.Bytes()))
	return buf.Bytes()
}

// TODO
func (tglp *TGLP_BFFNT) encodeSheets() []byte {
	return nil
}

func deswizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, data []byte) []byte {
	return swizzleSurface(width, height, depth, format, aa, use, tileMode, swizzle_, pitch, bpp, slice, sample, data, false)
}

func swizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, byte, data []byte) []byte {
	return swizzleSurface(width, height, depth, format, aa, use, tileMode, swizzle_, pitch, bpp, slice, sample, data, true)
}

// Copied from KillzXGaming/Switch-Toolbox
// KillzXGaming/Switch-Toolbox credits ____________
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

	if debug {
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

	if debug {
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

func (cmap *CMAP) decode(allRaw []byte, cmapOffset uint32) []CMAP {
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

	if debug {
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

	if debug {
		dataPosEnd := headerEnd + dataPos
		fmt.Printf("Read section total of %d bytes\n", dataPosEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header           %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("data calculated  %-8d to  %d\n", headerEnd, dataPosEnd)
		padding := headerStart + int(cmap.SectionSize) - dataPosEnd
		fmt.Printf("pad %d bytes     %-8d to  %d\n", padding, dataPosEnd, dataPosEnd+padding)
		fmt.Println()
	}

	return nil
}

func (cmap *CMAP) encode() []byte {
	return nil
}

func decodeAllCmaps(allRaw []byte, offset uint32) []CMAP {
	res := make([]CMAP, 0)

	for offset != 0 {
		var currentCmap CMAP
		currentCmap.decode(allRaw, offset)
		res = append(res, currentCmap)

		offset = currentCmap.NextCMAPOffset
	}

	return res
}

// This BFFNT file is Breath of the Wild's NormalS_00.bffnt. The goal of the
// project is to create a bffnt encoder/decoder so I can upscale this font

const (
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/Ancient/Ancient_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/Special/Special_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/Caption/Caption_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/Normal/Normal_00.bffnt"
	testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/NormalS/NormalS_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/External/External_00.bffnt"

	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/comicfont/Normal_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/kirbysans/Normal_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/kirbyscript/Normal_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/popjoy_font/Normal_00.bffnt"
	// testBffntFile = "/Users/kyeap/workspace/bffnt/WiiU_fonts/turbofont/Normal_00.bffnt"
)

func main() {
	flag.BoolVar(&debug, "d", false, "enable debug output")
	flag.Parse()

	bffntRaw, err := ioutil.ReadFile(testBffntFile)
	handleErr(err)

	var cfnt CFNT
	cfnt.decode(bffntRaw)
	_ = cfnt.encode()

	var finf FINF_BFFNT
	finf.decode(bffntRaw)
	_ = finf.encode()

	var tglp TGLP_BFFNT
	tglp.decode(bffntRaw)
	_ = tglp.encodeHeader()
	_ = tglp.encodeSheets()

	var cwdh CWDH
	cwdh.decode(bffntRaw, finf.CWDHOffset)
	_ = cwdh.encode()

	_ = decodeAllCmaps(bffntRaw, finf.CMAPOffset)

	//KERNING TABLES WIP
	// There are 3084 bytes left over

	// a kerning pair is made up of 2 characters and a kerning value.  it lets
	// us know how much two characters should be offsetted from each other for
	// a more aesthetically pleasing visual.
	//
	//
	// Offset  Size  Description
	// 0x00    0x04  Magic Header (KRNG)
	// 0x04    0x04  Section Size
	// 0x08    0x02  amount of First Chars
	// 0x0A    0x02  First char in a pair
	// 0x0C    0x02  Offset to the array of second characters (must multiply by 2)

	// When going to the second table then you read
	// 0x0E    0x02  amount of second characters
	// 0x10    0x02  second char in a pair
	// 0x12    0x02  kerning value

	pos := 536080 // KRNG start
	data := bffntRaw[pos:]

	dataPos := 0
	fmt.Println(string(data[0:4]))
	fmt.Printf("section size: %v\n", binary.BigEndian.Uint32(data[4:8]))
	firstCharCount := binary.BigEndian.Uint16(data[8:10])
	fmt.Printf("amount of FirstChars: %v\n", firstCharCount)

	dataPos += 10

	e := int(firstCharCount)
	for i := 0; i < e; i++ {
		firstChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		offset := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
		dataPos += 4

		fmt.Printf("( '%s', %d )\n", string(firstChar), offset)
	}
	// dataPos is 378

	fmt.Println("SECOND CHARS============================")
	// decode 2nd char?
	for i := 0; i < e; i++ {
		fmt.Println("data index?:", (dataPos-8)/2)
		secondCharCount := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		dataPos += 2
		fmt.Printf("amount of SecondChars: %v\n", secondCharCount)
		for i := 0; i < int(secondCharCount); i++ {
			secondChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			offset := int16(binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4]))
			dataPos += 4

			fmt.Printf("( '%s', %d )\n", string(secondChar), offset)
		}
	}

	fmt.Println(dataPos)

	// leftover bytes?
	leftover := uint16(binary.BigEndian.Uint16(data[dataPos : dataPos+2]))
	fmt.Println(leftover)

	return
}
