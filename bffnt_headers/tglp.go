package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

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

type TGLP struct { //    Offset  Size  Description
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
	SheetData        []image.NRGBA // separated unswizzled images
}

// Version 4 (BFFNT)
// The input for TGLP decode is the entire BFFNT file in the form of a byte
// array ([]byte).
func (tglp *TGLP) Decode(raw []byte) {
	headerStart := CFNT_HEADER_SIZE + FINF_HEADER_SIZE
	headerEnd := headerStart + TGLP_HEADER_SIZE
	headerRaw := raw[headerStart:headerEnd]
	assertEqual(TGLP_HEADER_SIZE, len(headerRaw))
	tglp.DecodeHeader(headerRaw)

	totalSheetDataSize := int(tglp.SheetSize) * int(tglp.NumOfSheets)
	dataStart := int(tglp.SheetDataOffset)
	dataEnd := dataStart + totalSheetDataSize
	tglp.AllSheetData = raw[dataStart:dataEnd]

	calculatedTGLPSectionSize := TGLP_HEADER_SIZE + tglp.computePredataPadding() + len(tglp.AllSheetData)
	assertEqual(int(tglp.SectionSize), calculatedTGLPSectionSize)

	tglp.DecodeSheets()
	if Debug {
		fmt.Printf("Read section total of %d bytes\n", dataEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header      %-8d to  %d\n", headerStart, headerEnd)
		fmt.Printf("padding     %-8d to  %d\n", headerEnd, dataStart)
		fmt.Printf("image data  %-8d to  %d\n", dataStart, dataEnd)
		fmt.Println()
	}
}

func (tglp *TGLP) DecodeHeader(raw []byte) {
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

	if Debug {
		pprint(tglp)
	}
}

// TODO: decode multiple sheets
func (tglp *TGLP) DecodeSheets() {
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
	img := imaging.FlipV(alphaImg.SubImage(alphaImg.Rect))

	tglp.SheetData = append(tglp.SheetData, *img)

	// f, err := os.Create("outimage.png")
	// handleErr(err)
	// defer f.Close()

	// // Encode to `PNG` with `DefaultCompression` level
	// // then save to file
	// err = png.Encode(f, alphaImg.SubImage(alphaImg.Rect))
	// handleErr(err)
}

func (tglp *TGLP) Encode() []byte {
	var res []byte

	header := tglp.EncodeHeader()
	padding := make([]byte, tglp.computePredataPadding())
	allSheetData := tglp.EncodeSheetData()

	res = append(res, header...)
	res = append(res, padding...)
	res = append(res, allSheetData...)

	assertEqual(int(tglp.SheetDataOffset), CFNT_HEADER_SIZE+FINF_HEADER_SIZE+TGLP_HEADER_SIZE+len(padding))
	assertEqual(int(tglp.SectionSize), len(res))
	return res
}

func (tglp *TGLP) EncodeHeader() []byte {
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

func (tglp *TGLP) computePredataPadding() int {
	// Not to scale representation of a portion of the bffnt file in raw bytes
	// for visual purposes
	//                      |-------------TGLP section size---------------------------|
	// CFNT   FINF          TGLP header    padding              tglp SheetDataOffset
	// |      |             |              |                    |
	// aaaaaa bbbbbbbbbbbbb cccccccccccccc 00000000000000000000 ddddddddddddddddddddddd

	return int(tglp.SheetDataOffset) - CFNT_HEADER_SIZE - FINF_HEADER_SIZE - TGLP_HEADER_SIZE
}

func (tglp *TGLP) EncodeSheetData() []byte {
	encodedSheetData := make([]byte, 0)

	// swizzle every sheet
	for i := 0; i < len(tglp.SheetData); i++ {
		currentSheet := tglp.SheetData[i]

		// Wii U stores image data upside down
		img := imaging.FlipV(currentSheet.SubImage(currentSheet.Rect))

		sheetData := make([]byte, tglp.SheetSize)
		switch tglp.SheetImageFormat {
		case 8:
			// convert RGBA into alpha only image, discard unused bytes
			for i := 0; i < len(sheetData); i++ {
				sheetData[i] = img.Pix[4*i+3]
			}
			break
		default:
			panic(fmt.Sprintf("Unsupported image encoding for image format: %d", tglp.SheetImageFormat))
		}

		// swizzle the image
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
		swizzledData := swizzle(sw, sh, depth, sh, format_, aa, use, tileMode, swizzle_, sw, bpp, slice, sample, sheetData)

		// write swizzled sheet
		encodedSheetData = append(encodedSheetData, swizzledData...)
	}

	return encodedSheetData
}

func deswizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, data []byte) []byte {
	return swizzleSurface(width, height, depth, format, aa, use, tileMode, swizzle_, pitch, bpp, slice, sample, data, false)
}

func swizzle(width uint, height uint, depth uint, height_ uint, format uint, aa uint, use uint, tileMode uint, swizzle_ uint, pitch uint, bpp uint, slice uint, sample uint, data []byte) []byte {
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
