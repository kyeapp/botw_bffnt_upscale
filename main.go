package main

import (
	"bffnt/bffnt_headers"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Resources
// https://www.3dbrew.org/wiki/BCFNT#Version_4_.28BFFNT.29
// http://wiki.tockdom.com/wiki/BRFNT_(File_Format)
// https://github.com/KillzXGaming/Switch-Toolbox/blob/12dfbaadafb1ebcd2e07d239361039a8d05df3f7/File_Format_Library/FileFormats/Font/BXFNT/FontKerningTable.cs

type BFFNT struct {
	CFNT  bffnt_headers.CFNT
	FINF  bffnt_headers.FINF
	TGLP  bffnt_headers.TGLP
	CWDHs []bffnt_headers.CWDH
	CMAPs []bffnt_headers.CMAP
	KRNG  bffnt_headers.KRNG
}

var bffntRaw []byte
var err error

func (b *BFFNT) Decode(bffntRaw []byte) {
	b.CFNT.Decode(bffntRaw)
	b.FINF.Decode(bffntRaw)
	b.TGLP.Decode(bffntRaw)
	b.CWDHs = bffnt_headers.DecodeCWDHs(bffntRaw, b.FINF.CWDHOffset)
	b.CMAPs = bffnt_headers.DecodeCMAPs(bffntRaw, b.FINF.CMAPOffset)
	b.KRNG.Decode(bffntRaw)
}

func (b *BFFNT) Encode() []byte {
	res := make([]byte, 0)

	tglpRaw := b.TGLP.Encode()

	cwdhStartOffset := bffnt_headers.CFNT_HEADER_SIZE + bffnt_headers.FINF_HEADER_SIZE + len(tglpRaw)
	cwdhsRaw := bffnt_headers.EncodeCWDHs(b.CWDHs, cwdhStartOffset)

	cmapStartOffset := cwdhStartOffset + len(cwdhsRaw)
	cmapsRaw := bffnt_headers.EncodeCMAPs(b.CMAPs, cmapStartOffset)

	krngRaw := b.KRNG.Encode()

	tglpOffset := bffnt_headers.CFNT_HEADER_SIZE + bffnt_headers.FINF_HEADER_SIZE
	cwdhOffset := tglpOffset + len(tglpRaw)
	cmapOffset := cwdhOffset + len(cwdhsRaw)
	finfRaw := b.FINF.Encode(tglpOffset+8, cwdhOffset+8, cmapOffset+8)

	// TODO: calculate an appriopriate blockreadnum based on sheetsize?
	fileSize := uint32(bffnt_headers.CFNT_HEADER_SIZE + len(finfRaw) + len(tglpRaw) + len(cwdhsRaw) + len(cmapsRaw) + len(krngRaw))
	cfntRaw := b.CFNT.Encode(fileSize)

	res = append(res, cfntRaw...)
	res = append(res, finfRaw...)
	res = append(res, tglpRaw...)
	res = append(res, cwdhsRaw...)
	res = append(res, cmapsRaw...)
	res = append(res, krngRaw...)

	return res
}

// This is to be used to upscale the resolution of the a texture. It will make
// the appropriate calculations based on the amount of scaling specified
// It will be up to the user to provide the upscaled images in a png format
func (b *BFFNT) Upscale(scale uint8) {
	fmt.Println("upscaling image by factor of", scale)
	// TODO: Instead of an integer scaler. change this to be a ratio. you could
	// then do gradient scaling.  e.x. scale by 1.5x

	b.FINF.Upscale(scale)
	b.TGLP.Upscale(scale)

	for i, _ := range b.CWDHs {
		b.CWDHs[i].Upscale(scale)
	}

	b.KRNG.Upscale(scale)

}

// This BFFNT file is Breath of the Wild's NormalS_00.bffnt. The goal of the
// project is to create a bffnt encoder/decoder so I can upscale this font

const (
	// testBffntFile = "./WiiU_fonts/botw/Ancient/Ancient_00.bffnt"
	// testBffntFile = "./WiiU_fonts/botw/Special/Special_00.bffnt"
	testBffntFile = "./WiiU_fonts/botw/Caption/Caption_00.bffnt"
	// testBffntFile = "./WiiU_fonts/botw/Normal/Normal_00.bffnt"
	// testBffntFile = "./WiiU_fonts/botw/NormalS/NormalS_00.bffnt"
	// testBffntFile = "./WiiU_fonts/botw/External/External_00.bffnt"

	// testBffntFile = "./WiiU_fonts/comicfont/Normal_00.bffnt"
	// testBffntFile = "./WiiU_fonts/kirbysans/Normal_00.bffnt"
	// testBffntFile = "./WiiU_fonts/kirbyscript/Normal_00.bffnt"
	// testBffntFile = "./WiiU_fonts/popjoy_font/Normal_00.bffnt"
	// testBffntFile = "./WiiU_fonts/turbofont/Normal_00.bffnt"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.BoolVar(&bffnt_headers.Debug, "d", false, "enable debug output")
	flag.Parse()

	bffntRaw, err = ioutil.ReadFile(testBffntFile)

	var bffnt BFFNT
	handleErr(err)
	bffnt.Decode(bffntRaw)

	// this upscales the character width height and kerning tables.
	// the images are blank.
	bffnt.Upscale(2)

	encodedRaw := bffnt.Encode()

	err = os.WriteFile("template.bffnt", encodedRaw, 0644)
	handleErr(err)

	// bffnt.Decode(encodedRaw)

	generateTexture(bffnt)

	return
}

func pprint(s interface{}) {
	jsonBytes, err := json.MarshalIndent(s, "", "  ")
	handleErr(err)

	fmt.Printf("%s\n", string(jsonBytes))
}

// https://pkg.go.dev/golang.org/x/image/font/sfnt#Font
func generateTexture(b BFFNT) {
	pairSlice := make([]bffnt_headers.AsciiIndexPair, 0)
	for _, cmap := range b.CMAPs {
		for j, _ := range cmap.CharAscii {
			if cmap.CharIndex[j] != 65535 {
				p := bffnt_headers.AsciiIndexPair{
					CharAscii: cmap.CharAscii[j],
					CharIndex: cmap.CharIndex[j],
				}

				// fmt.Printf("(%d, %s), ", p.CharIndex, string(p.CharAscii))
				pairSlice = append(pairSlice, p)
			}
		}
	}

	sort.Slice(pairSlice, func(i, j int) bool {
		return pairSlice[i].CharIndex < pairSlice[j].CharIndex
	})

	fmt.Printf("%d characters indexed\n", len(pairSlice))

	var (
		// these are the original pixel counts meant for
		// scale 1 for 1280×720
		// scale 2 for 2560 × 1440
		// scale 3 for 3840 x 2160
		scale = 2

		// filename       = fmt.Sprintf("Normal_00_%dx.png", scale)
		// fontSize       = 45 // 4k
		// fontSize = 32 - xOffset // 2k

		filename = fmt.Sprintf("Caption_00_%dx.png", scale)
		fontSize = 9 * scale
		xOffset  = 0

		// filename = fmt.Sprintf("NormalS_00_%dx.png", scale)
		// fontSize = 10 * scale
		// xOffset  = 2 * scale

		cellWidth   = int(b.TGLP.CellWidth)
		cellHeight  = int(b.TGLP.CellHeight)
		columnCount = int(b.TGLP.NumOfColumns)
		baseline    = int(b.TGLP.BaselinePosition) + scale
		sheetHeight = int(b.TGLP.SheetHeight)
		sheetWidth  = int(b.TGLP.SheetWidth)

		// every cell is separated by 1 px length padding at the left and top.
		realBaseline   = baseline + 1
		realCellWidth  = cellWidth + 1
		realCellHeight = cellHeight + 1
	)

	dat, err := os.ReadFile("./FOT-RodinNTLGPro-DB.BFOTF.otf")
	handleErr(err)

	f, err := opentype.Parse(dat)
	handleErr(err)

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(fontSize),
		DPI:     144,
		Hinting: font.HintingFull,
	})
	handleErr(err)

	// drawer.MeasureString can be used to modify kerning table
	dst := image.NewAlpha(image.Rect(0, 0, sheetWidth, sheetHeight))
	glyphDrawer := font.Drawer{
		Dst:  dst,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(0, 0),
	}

	var charIndex, x, y int
	for rowIndex := 0; rowIndex < 9999; rowIndex++ {
		y = realCellHeight*rowIndex + realBaseline
		for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
			x = realCellWidth * columnIndex
			glyphDrawer.Dot = fixed.P(x, y)
			fmt.Printf("The dot is at %v\n", glyphDrawer.Dot)

			glyph := string(pairSlice[charIndex].CharAscii)
			glyphBoundAtDot, _ := glyphDrawer.BoundString(glyph)
			// fmt.Println(x, glyphBoundAtDot.Min.X, glyphBoundAtDot.Min.Y, glyphBoundAtDot.Max.X, glyphBoundAtDot.Max.Y)

			// calculate glyph x offset in it's cell so that there is only 1
			// pixel length between the cell and the left most pixel of the
			// glyph we are abount to draw. Generally the characters are draw
			// to the right of the Dot but its possible for this to be
			// negative. e.x. character j's left most pixel falls to the left
			// of the dot.
			leftAlignOffset := int(glyphBoundAtDot.Min.X/64) - x
			// fmt.Println(leftAlignOffset)

			// Use this to calculate kerning

			y_nintendo := y - scale // manual adjust to compensate difference between nintendo font generator and mine.
			glyphDrawer.Dot = fixed.P(x-leftAlignOffset+(xOffset)+1, y_nintendo)
			glyphDrawer.DrawString(glyph)

			// Alight character left

			charIndex++

			// exit when no more characters
			if charIndex == len(pairSlice) {
				goto writePng
			}
		}
	}

writePng:
	// draw grid lines. Good for debugging.
	for x := 0; x < int(b.TGLP.SheetWidth); x += realCellWidth {
		VLine(dst, x, 0, int(b.TGLP.SheetHeight)) // draw column
	}
	for y := 0; y < int(b.TGLP.SheetHeight); y += realCellHeight {
		HLine(dst, 0, y, int(b.TGLP.SheetWidth)) // draw row
	}
	for y := int(b.TGLP.BaselinePosition) + 1; y < int(b.TGLP.SheetHeight); y += realCellHeight {
		HLine(dst, 0, y, int(b.TGLP.SheetWidth)) // draw row
	}

	_ = os.Remove(filename)

	fmt.Println("wrote glyphs to", filename)
	textureFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	handleErr(err)
	err = png.Encode(textureFile, dst)
	handleErr(err)
}

// HLine draws a horizontal line
func HLine(img *image.Alpha, x1, y, x2 int) {
	for ; x1 <= x2; x1++ {
		// if img.At(x1, y) != color.Transparent {
		// 	fmt.Printf("%d, %d\n", x1, y)
		// 	panic("character crosses gridline")
		// }
		if bffnt_headers.Debug {
			img.Set(x1, y, color.Opaque)
		}
	}
}

// VLine draws a veritcal line
func VLine(img *image.Alpha, x, y1, y2 int) {
	for ; y1 <= y2; y1++ {
		if bffnt_headers.Debug {
			img.Set(x, y1, color.Opaque)
		}
	}
}
