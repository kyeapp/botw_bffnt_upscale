package main

import (
	"flag"
	"io/ioutil"

	"bffnt/bffnt_headers"
)

type BFFNT struct {
	CFNT  bffnt_headers.CFNT
	FINF  bffnt_headers.FINF
	TGLP  bffnt_headers.TGLP
	CWDHs []bffnt_headers.CWDH
	CMAPs []bffnt_headers.CMAP
	KRNG  bffnt_headers.KRNG
}

func (b *BFFNT) Load(bffntFile string) {
	bffntRaw, err := ioutil.ReadFile(bffntFile)
	if err != nil {
		panic(err)
	}

	b.CFNT.Decode(bffntRaw)
	// _ = cfnt.encode()

	b.FINF.Decode(bffntRaw)
	// _ = finf.encode()

	b.TGLP.Decode(bffntRaw)
	// _ = tglp.encodeHeader()
	// _ = tglp.encodeSheets()

	// b.cwdhs.Decode(bffntRaw, b.finf.CWDHOffset)
	// _ = cwdh.encode()

	b.CMAPs = bffnt_headers.DecodeAllCmaps(bffntRaw, b.FINF.CMAPOffset)

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

	flag.BoolVar(&bffnt_headers.Debug, "d", false, "enable debug output")
	flag.Parse()

	var b BFFNT
	b.Load(testBffntFile)

	return
}
