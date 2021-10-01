package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	BOTW_NormalS test_bffnt
)

func defineTestBFFNTs() {
	// Breath of the Wild v1.6.0
	var b BFFNT
	BOTW_NormalS = b.Load("/Users/kyeap/workspace/bffnt/WiiU_fonts/botw/NormalS/NormalS_00.bffnt")

	// BOTW_NormalS = test_bffnt{
	// 	cfnt:       []byte{70, 70, 78, 84, 254, 255, 0, 20, 3, 0, 0, 0, 0, 8, 58, 28, 0, 9, 0, 0},
	// 	finf:       []byte{70, 73, 78, 70, 0, 0, 0, 32, 2, 30, 24, 23, 0, 30, 0, 0, 0, 24, 24, 1, 0, 0, 0, 60, 0, 8, 32, 8, 0, 8, 39, 64},
	// 	tglpHeader: []byte{84, 71, 76, 80, 0, 8, 31, 204, 24, 30, 1, 21, 0, 8, 0, 0, 0, 23, 0, 8, 0, 20, 0, 33, 2, 0, 4, 0, 0, 0, 32, 0},
	// }
}

// decode the test cases and encode them again. Compares the bytes that get
// encoded and them decoded. They should be the same.

func TestCFNT(t *testing.T) {
	expected := BOTW_NormalS.cfnt
	var testCFNT CFNT
	testCFNT.decode(expected)
	actual := testCFNT.encode()

	assert.Equal(t, expected, actual)
}

func TestFINF(t *testing.T) {
	expected := BOTW_NormalS.finf

	var testFINF FINF_BFFNT
	testFINF.decode(expected)
	actual := testFINF.encode()

	assert.Equal(t, expected, actual)
}

func TestTGLP(t *testing.T) {
	expected := BOTW_NormalS.tglpHeader

	var testTGLP TGLP_BFFNT
	testTGLP.decodeHeader(expected)
	actual := testTGLP.encodeHeader()

	assert.Equal(t, expected, actual)

	//test decodeSheets
}

func TestMain(m *testing.M) {
	defineTestBFFNTs()
	code := m.Run()
	os.Exit(code)
}
