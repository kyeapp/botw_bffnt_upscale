package main

import (
	"bffnt/bffnt_headers"
	"flag"
	"fmt"
	"image"
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
	// testBffntFile = "./WiiU_fonts/botw/Caption/Caption_00.bffnt"
	// testBffntFile = "./WiiU_fonts/botw/Normal/Normal_00.bffnt"
	testBffntFile = "./WiiU_fonts/botw/NormalS/NormalS_00.bffnt"
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
	bffnt.Upscale(3)

	encodedRaw := bffnt.Encode()

	err = os.WriteFile("output.bffnt", encodedRaw, 0644)
	handleErr(err)

	// bffnt.Decode(encodedRaw)

	generateTexture(bffnt)

	return
}

func generateTexture(b BFFNT) {
	pairSlice := make([]bffnt_headers.AsciiIndexPair, 0)
	for _, cmap := range b.CMAPs {
		for j, _ := range cmap.CharAscii {
			if cmap.CharIndex[j] != 65535 {
				p := bffnt_headers.AsciiIndexPair{
					CharAscii: cmap.CharAscii[j],
					CharIndex: cmap.CharIndex[j],
				}
				pairSlice = append(pairSlice, p)
			}
		}
	}
	fmt.Println(pairSlice)
	sort.Slice(pairSlice, func(i, j int) bool {
		return pairSlice[i].CharIndex < pairSlice[i].CharIndex
	})
	fmt.Println(pairSlice)

	const (
		scale = 3
		// attributes of NormalS_00.bffnt
		cellWidth   = 24 * scale
		cellHeight  = 30 * scale
		columnCount = 20
		rowCount    = 33

		baseLine    = 23 * scale // ascent
		sheetHeight = 1024 * scale
		sheetWidth  = 512 * scale

		realCellWidth  = cellWidth + 1
		realCellHeight = cellHeight + 1

		fontSize = 10
	)

	dat, err := os.ReadFile("./FOT-RodinNTLGPro-DB.BFOTF.otf")
	handleErr(err)

	// f, err := opentype.Parse(goitalic.TTF)
	f, err := opentype.Parse(dat)
	handleErr(err)

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    fontSize * scale,
		DPI:     144,
		Hinting: font.HintingNone,
	})
	handleErr(err)

	dst := image.NewGray(image.Rect(0, 0, sheetWidth, sheetHeight))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(0, 0),
	}

	var k, x, y int

	fmt.Println(len(pairSlice))

	for i := 0; i < rowCount; i++ {
		x = 1
		y = realCellHeight * i
		for j := 0; j < columnCount; j++ {
			x = realCellWidth * j
			d.Dot = fixed.P(x, y+baseLine)
			// fmt.Printf("The dot is at %v\n", d.Dot)

			d.DrawString(string(pairSlice[k].CharAscii))
			k++

			if k == len(pairSlice) {
				goto writePng
			}
		}
	}

writePng:
	ff, err := os.OpenFile("test.png", os.O_CREATE|os.O_RDWR, 0644)
	handleErr(err)
	err = png.Encode(ff, dst)
	handleErr(err)
}
