package main

import (
	"bffnt/bffnt_headers"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"

	"github.com/disintegration/imaging"
)

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
func (b *BFFNT) Upscale(scale uint8, upscaledImages []image.NRGBA) {
	fmt.Println("upscaling image by factor of", scale)
	// TODO: Instead of an integer scaler. change this to be a ratio. you could
	// then do gradient scaling.  e.x. scale by 1.5x

	b.FINF.Upscale(scale)
	b.TGLP.Upscale(scale, upscaledImages)
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

	upscaledReader, err := os.Open("./sheet_0_waifu2x.png")
	handleErr(err)
	img, err := png.Decode(upscaledReader)
	handleErr(err)

	g := img.(*image.Gray)
	fmt.Printf("image type: %T\n", g)

	// I'm just using the flip function to convert to NRGBA
	nrgbaFlipped := imaging.FlipV(g.SubImage(g.Rect))
	sheet0 := imaging.FlipV(nrgbaFlipped.SubImage(nrgbaFlipped.Rect))

	upscaledSheets := make([]image.NRGBA, 0)
	upscaledSheets = append(upscaledSheets, *sheet0)

	bffnt.Upscale(2, upscaledSheets)

	encodedRaw := bffnt.Encode()

	err = os.WriteFile("output.bffnt", encodedRaw, 0644)
	handleErr(err)

	bffnt.Decode(encodedRaw)

	return
}
