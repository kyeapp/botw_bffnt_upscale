package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	const (
		scale = 3
		// attributes of NormalS_00.bffnt
		cellWidth  = 24 * scale
		cellHeight = 30 * scale

		sheetHeight = cellHeight * 2
		sheetWidth  = 512 * scale
	)
	var (
		x = 1
		y = 23 * scale // ascent
	)

	dat, err := os.ReadFile("./FOT-RodinNTLGPro-DB.BFOTF.otf")
	check(err)

	// f, err := opentype.Parse(goitalic.TTF)
	f, err := opentype.Parse(dat)
	if err != nil {
		log.Fatalf("Parse: %v", err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    10 * scale,
		DPI:     144,
		Hinting: font.HintingNone,
	})
	if err != nil {
		log.Fatalf("NewFace: %v", err)
	}

	dst := image.NewGray(image.Rect(0, 0, sheetWidth, sheetHeight))
	d := font.Drawer{
		Dst:  dst,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	fmt.Printf("The dot is at %v\n", d.Dot)
	d.DrawString(`    !   "  #  $  %  & '  (   )  *   +  ,  -  .  /  0  1  2  3`)
	fmt.Printf("The dot is at %v\n", d.Dot)
	x = 1
	y += cellHeight + 1
	d.Dot = fixed.P(x, y)
	fmt.Printf("The dot is at %v\n", d.Dot)
	d.DrawString(`4  5  6  7  8  9  :  ;  <  =  >  ?  @  A  B  C  D  E  F  G`)
	fmt.Printf("The dot is at %v\n", d.Dot)

	ff, err := os.OpenFile("test.png", os.O_CREATE|os.O_RDWR, 0644)
	check(err)
	err = png.Encode(ff, dst)
	check(err)
}
