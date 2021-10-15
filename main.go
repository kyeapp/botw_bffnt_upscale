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
// https://torinak.com/font/lsfont.html
// https://www.dafont.com/botw-hylian.font

type BFFNT struct {
	FFNT  bffnt_headers.FFNT
	FINF  bffnt_headers.FINF
	TGLP  bffnt_headers.TGLP
	CWDHs []bffnt_headers.CWDH
	CMAPs []bffnt_headers.CMAP
	KRNG  bffnt_headers.KRNG
}

var bffntRaw []byte
var err error

func (b *BFFNT) Decode(bffntRaw []byte) {
	b.FFNT.Decode(bffntRaw)
	b.FINF.Decode(bffntRaw)
	b.TGLP.Decode(bffntRaw)
	b.CWDHs = bffnt_headers.DecodeCWDHs(bffntRaw, b.FINF.CWDHOffset)
	b.CMAPs = bffnt_headers.DecodeCMAPs(bffntRaw, b.FINF.CMAPOffset)
	b.KRNG.Decode(bffntRaw)
}

func (b *BFFNT) Encode() []byte {
	tglpOffset := bffnt_headers.FFNT_HEADER_SIZE + bffnt_headers.FINF_HEADER_SIZE + 8
	tglpRaw := b.TGLP.Encode()

	cwdhOffset := tglpOffset + len(tglpRaw)
	cwdhsRaw := bffnt_headers.EncodeCWDHs(b.CWDHs, cwdhOffset)

	cmapOffset := cwdhOffset + len(cwdhsRaw)
	cmapsRaw := bffnt_headers.EncodeCMAPs(b.CMAPs, cmapOffset)

	finfRaw := b.FINF.Encode(tglpOffset, cwdhOffset, cmapOffset)

	krngOffset := cmapOffset + len(cmapsRaw)
	krngRaw := b.KRNG.Encode(uint32(krngOffset))

	// TODO: calculate an appriopriate blockreadnum based on sheetsize?
	fileSize := uint32(bffnt_headers.FFNT_HEADER_SIZE + len(finfRaw) + len(tglpRaw) + len(cwdhsRaw) + len(cmapsRaw) + len(krngRaw))
	ffntRaw := b.FFNT.Encode(fileSize)

	res := make([]byte, 0)
	res = append(res, ffntRaw...)
	res = append(res, finfRaw...)
	res = append(res, tglpRaw...)
	res = append(res, cwdhsRaw...)
	res = append(res, cmapsRaw...)
	res = append(res, krngRaw...)

	return res
}

func pprint(s interface{}) {
	jsonBytes, err := json.MarshalIndent(s, "", "  ")
	// jsonBytes, err := json.Marshal(s)
	handleErr(err)

	fmt.Printf("%s\n", string(jsonBytes))
}

// This is to be used to upscale the resolution of the a texture. It will make
// the appropriate calculations based on the amount of scaling specified
// It will be up to the user to provide the upscaled images in a png format
func (b *BFFNT) Upscale(scale uint8) {
	fmt.Println("upscaling image by factor of", scale)
	// TODO: Instead of an integer scaler. change this to be a ratio. you could
	// then do gradient scaling.  e.x. scale by 1.5x

	// testing
	// f := b.FINF
	// b.FINF.Upscale(1)
	// if !reflect.DeepEqual(f, b.FINF) {
	// 	pprint(f)
	// 	pprint(b.FINF)
	// 	panic("FINF bad")
	// }

	// b.TGLP.Print()
	// b.TGLP.Upscale(1)
	// b.TGLP.Print()

	// for i, _ := range b.CWDHs {
	// 	c := b.CWDHs[i]
	// 	b.CWDHs[i].Upscale(1)
	// 	if !reflect.DeepEqual(c, b.CWDHs[i]) {
	// 		panic("CWDH bad")
	// 	}
	// }

	// k := b.KRNG
	// b.KRNG.Upscale(1)
	// if !reflect.DeepEqual(k, b.KRNG) {
	// 	panic("KRNG bad")
	// }

	// panic("debug stop")

	b.FINF.Upscale(scale)
	b.TGLP.Upscale(scale)

	for i, _ := range b.CWDHs {
		b.CWDHs[i].Upscale(scale)
	}

	b.KRNG.Upscale(scale)
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.BoolVar(&bffnt_headers.Debug, "d", false, "enable debug output")
	flag.Parse()

	// scale 1 for 1280×720 (original)
	// scale 2 for 2560 × 1440
	// scale 3 for 3840 x 2160
	scale := 2

	// upscaleBffnt("Ancient", "./nintendo_system_ui/botw-sheikah.ttf", scale)
	// upscaleBffnt("Caption", "./nintendo_system_ui/DSi-Wii-3DS-Wii_U/FOT-RodinBokutoh-Pro-M.otf", scale)
	upscaleBffnt("Normal", "./nintendo_system_ui/DSi-Wii-3DS-Wii_U/FOT-RodinBokutoh-Pro-B.otf", scale)
	// upscaleBffnt("NormalS", "./nintendo_system_ui/DSi-Wii-3DS-Wii_U/FOT-RodinBokutoh-Pro-DB.otf", 2)
	// upscaleBffnt("External", "./nintendo_system_ui/nintendo_ext_003.ttf", scale)

	return
}

func upscaleBffnt(botwFontName string, fontFile string, scale int) {
	bffntFile := fmt.Sprintf("./WiiU_fonts/botw/%[1]s/%[1]s_00.bffnt", botwFontName)
	fmt.Println("Reading bffnt file", bffntFile)
	bffntRaw, err = ioutil.ReadFile(bffntFile)

	var bffnt BFFNT
	handleErr(err)
	bffnt.Decode(bffntRaw)

	// this upscales the character width height and kerning tables.
	// the images are blank.
	bffnt.Upscale(uint8(scale))

	encodedRaw := bffnt.Encode()
	fmt.Println("encoded bytes:", len(encodedRaw))

	outputBffntFile := fmt.Sprintf("%s_00_%dx_template.bffnt", botwFontName, scale)
	err = os.WriteFile(outputBffntFile, encodedRaw, 0644)
	handleErr(err)

	bffnt.Decode(encodedRaw)
	// panic("need to know krng decode is working")

	generateTexture(bffnt, botwFontName, fontFile, scale)
}

// https://pkg.go.dev/golang.org/x/image/font/sfnt#Font
func generateTexture(b BFFNT, fontName string, fontFile string, scale int) {
	pairSlice := make([]bffnt_headers.AsciiIndexPair, 0)
	for _, cmap := range b.CMAPs {
		for j, _ := range cmap.CharAscii {
			if cmap.CharIndex[j] != 65535 {
				p := bffnt_headers.AsciiIndexPair{
					CharAscii: cmap.CharAscii[j],
					CharIndex: cmap.CharIndex[j],
				}

				// fmt.Printf("(%d, %s)\n", p.CharIndex, string(p.CharAscii))
				// fmt.Printf("(%d, %d)\n", p.CharIndex, p.CharAscii)
				pairSlice = append(pairSlice, p)
			}
		}
	}

	sort.Slice(pairSlice, func(i, j int) bool {
		return pairSlice[i].CharIndex < pairSlice[j].CharIndex
	})

	fmt.Printf("%d characters indexed\n", len(pairSlice))

	fontSize, outlineOffset := getBotwFontSettings(fontName, scale)

	// Caption
	var (
		filename    = fmt.Sprintf("%s_00_%dx.png", fontName, scale)
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

	fmt.Println("Reading font file", fontFile)
	dat, err := os.ReadFile(fontFile)
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

	// play in normal mode
	// fmt.Println(face.Kern(rune('L'), rune('T')))

	var charIndex, x, y int
	for rowIndex := 0; rowIndex < 9999; rowIndex++ {
		y = realCellHeight*rowIndex + realBaseline
		for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
			x = realCellWidth * columnIndex
			glyphDrawer.Dot = fixed.P(x, y)
			// fmt.Printf("The dot is at %v\n", glyphDrawer.Dot)

			ascii := pairSlice[charIndex].CharAscii
			glyph := string(asciiToGlyph(fontName, ascii))
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

			y_nintendo := y - scale // manual adjust to compensate y difference between nintendo font generator and mine.
			glyphDrawer.Dot = fixed.P(x-leftAlignOffset+(outlineOffset)+1, y_nintendo)
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
	if bffnt_headers.Debug {
		// draw grid lines. Good for debugging.
		for x := 0; x < int(b.TGLP.SheetWidth); x += realCellWidth {
			drawVerticalLine(dst, x, 0, int(b.TGLP.SheetHeight)) // draw columns
		}
		for y := 0; y < int(b.TGLP.SheetHeight); y += realCellHeight {
			drawHorizontalLine(dst, 0, y, int(b.TGLP.SheetWidth)) // draw rows
		}
		for y := int(b.TGLP.BaselinePosition) + 1; y < int(b.TGLP.SheetHeight); y += realCellHeight {
			drawHorizontalLine(dst, 0, y, int(b.TGLP.SheetWidth)) // draw baseline
		}
	}

	_ = os.Remove(filename)

	fmt.Println("wrote glyphs to", filename)
	textureFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	handleErr(err)
	err = png.Encode(textureFile, dst)
	handleErr(err)
}

// Manual adjustments for each font to closely resemble the original
func getBotwFontSettings(fontName string, scale int) (fontSize int, outlineOffset int) {
	switch fontName {
	case "Ancient":
		fontSize = 6 * scale
		outlineOffset = 0

	case "Caption":
		fontSize = 9 * scale
		outlineOffset = 0

	case "Normal":
		fontSize = 15 * scale // 2k
		outlineOffset = 0

	case "NormalS":
		fontSize = 12 * scale
		outlineOffset = 2 * scale // hNormalS Characters will need a 3px wide outline with 20% opacaity. I use GIMP.

	case "External":
		fontSize = 15 * scale
		outlineOffset = 0

	default:
		panic("file texture generation settings unknown")
	}

	return
}

// In most cases the ascii code maps to the correct glyph in the font file. For
// some glyphs, the ascii does not match the glyph in the font file (because we
// don't have the exact font file nintendo used). If the font file stil has the
// correct glyph at a different index we can create a manual mapping here.  No
// manual mapping means the ascii maps to the correct index in the font file.
func asciiToGlyph(fontName string, ascii uint16) uint16 {
	var asciiToGlyph map[uint16]uint16
	switch fontName {
	case "Ancient":
	case "Caption":
	case "Normal":
	case "NormalS":
	case "External":
		asciiToGlyph = getBotwExternalMap()
	default:
		panic("unknown font mapping")
	}

	glyphIndex, manualMappingExists := asciiToGlyph[ascii]
	if manualMappingExists {
		return glyphIndex
	}

	return ascii
}

// mapping botw external font character indexes to nintendo_ext_003.ttf
func getBotwExternalMap() map[uint16]uint16 {
	botwExternalMapping := make(map[uint16]uint16, 0)

	botwExternalMapping[57408] = 57568 // A
	botwExternalMapping[57409] = 57569 // B
	botwExternalMapping[57410] = 57570 // X
	botwExternalMapping[57411] = 57571 // Y
	botwExternalMapping[57412] = 57572 // L
	botwExternalMapping[57413] = 57573 // R
	botwExternalMapping[57414] = 57574 // ZL
	botwExternalMapping[57415] = 57575 // ZR
	botwExternalMapping[57416] = 57587 // Power
	botwExternalMapping[57417] = 57616 // D-pad
	botwExternalMapping[57418] = 57588 // Home
	botwExternalMapping[57419] = 57583 // +
	botwExternalMapping[57420] = 57584 // -

	botwExternalMapping[57424] = 57473 // Ljoy down
	botwExternalMapping[57425] = 57474 // Rjoy down
	botwExternalMapping[57426] = 57473 // Ljoy up
	botwExternalMapping[57427] = 57474 // Rjoy up
	botwExternalMapping[57428] = 57473 // Ljoy left-right
	botwExternalMapping[57429] = 57474 // Rjoy left-right
	botwExternalMapping[57430] = 57473 // Ljoy press-down
	botwExternalMapping[57431] = 57474 // Rjoy press-down
	botwExternalMapping[57432] = 57473 // Ljoy right
	botwExternalMapping[57433] = 57474 // Rjoy right
	botwExternalMapping[57434] = 57473 // Ljoy left
	botwExternalMapping[57435] = 57473 // Rjoy left
	botwExternalMapping[57437] = 57473 // Rjoy up-down
	botwExternalMapping[57438] = 57473 // Ljoy
	botwExternalMapping[57439] = 57473 // Rjoy
	botwExternalMapping[57440] = 0     // D-pad up
	botwExternalMapping[57441] = 0     // D-pad down
	botwExternalMapping[57442] = 0     // D-pad left
	botwExternalMapping[57443] = 0     // D-pad right
	botwExternalMapping[57444] = 0     // D-pad up-down
	botwExternalMapping[57445] = 0     // D-pad left-right
	// (34, 57446)
	// (35, 57447)
	// (36, 57475)
	// (37, 57476)
	// (38, 57477)
	// (39, 57478)
	// (40, 57479)
	// (41, 57480)
	// (42, 57481)
	// (43, 57482)
	// (44, 57483)
	// (45, 57484)
	// (46, 57485)
	// (47, 57486)
	// (48, 57487)

	return botwExternalMapping
}

func drawHorizontalLine(img *image.Alpha, x1, y, x2 int) {
	for ; x1 <= x2; x1++ {
		img.Set(x1, y, color.Opaque)
	}
}

func drawVerticalLine(img *image.Alpha, x, y1, y2 int) {
	for ; y1 <= y2; y1++ {
		img.Set(x, y1, color.Opaque)
	}
}
