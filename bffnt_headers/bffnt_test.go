package bffnt_headers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBFFNT(t *testing.T) {
	// testCase(t, "../WiiU_fonts/botw/Ancient/Ancient_00.bffnt", "bc6525a0089b9ddc90a2f25a1d68291e")
	testCase(t, "../WiiU_fonts/botw/Special/Special_00.bffnt", "4d973f84b287d787e5b1ed8d1fd82799")
	testCase(t, "../WiiU_fonts/botw/Caption/Caption_00.bffnt", "efc0070d11289b18f28525a755e75acb")
	testCase(t, "../WiiU_fonts/botw/Normal/Normal_00.bffnt", "8d7f1ec5872da263a95a5937ccd8a372")
	testCase(t, "../WiiU_fonts/botw/NormalS/NormalS_00.bffnt", "f993a5822f3ce05e51e0440b46bd1345")
	testCase(t, "../WiiU_fonts/botw/External/External_00.bffnt", "1ccd353cceda991d51c156fbb8b8a891")

	testCase(t, "../WiiU_fonts/comicfont/Normal_00.bffnt", "f67eaccca824952de8cd26bb05db530b")
	testCase(t, "../WiiU_fonts/kirbysans/Normal_00.bffnt", "76c3b7edaed85fec14e0a195fc7dbdaa")
	testCase(t, "../WiiU_fonts/kirbyscript/Normal_00.bffnt", "a948720350878355009a364c3ff6206c")
	testCase(t, "../WiiU_fonts/popjoy_font/Normal_00.bffnt", "8c5bd5e7dd1d8eb0e17144ba4275c4b1")
	testCase(t, "../WiiU_fonts/turbofont/Normal_00.bffnt", "7d935c25fc18d26a5f4a6c2b5cf24cce")
}

func testCase(t *testing.T, bffntFile string, expectedFileHash string) {
	t.Log(fmt.Sprintf("Testing %s", bffntFile))
	bffntRaw, err := ioutil.ReadFile(bffntFile)
	handleErr(err)

	verifyBffnt(t, bffntRaw)

	// Verify bffnt file with a md5 hash
	hash, err := hash_file_md5(bffntFile)
	handleErr(err)
	assert.Equal(t, expectedFileHash, hash, "md5 hash of bffnt file mismatch. test is invalid.")

	var ffnt FFNT
	ffnt.Decode(bffntRaw)
	encodedFFNT := ffnt.Encode(ffnt.TotalFileSize)
	expectedFFNT := bffntRaw[:FFNT_HEADER_SIZE]
	assert.Equal(t, expectedFFNT, encodedFFNT, "FFNT encoding did not produce the correct results")

	var finf FINF
	finf.Decode(bffntRaw)
	encodedFINF := finf.Encode(int(finf.TGLPOffset), int(finf.CWDHOffset), int(finf.CMAPOffset))
	expectedFINF := bffntRaw[FFNT_HEADER_SIZE : FFNT_HEADER_SIZE+FINF_HEADER_SIZE]
	assert.Equal(t, expectedFINF, encodedFINF, "FINF encoding did not produce the correct results")

	var tglp TGLP
	tglpHeaderStart := FFNT_HEADER_SIZE + FINF_HEADER_SIZE
	tglpHeaderEnd := tglpHeaderStart + TGLP_HEADER_SIZE
	tglp.DecodeHeader(bffntRaw[tglpHeaderStart:tglpHeaderEnd])
	encodedTGLPHeader := tglp.EncodeHeader()
	expectedTGLPHeader := bffntRaw[tglpHeaderStart:tglpHeaderEnd]
	assert.Equal(t, expectedTGLPHeader, encodedTGLPHeader, "TGLP Header encoding did not produce the correct results")
	encodedTGLP := tglp.Encode()
	// check data length is correct at least
	tglpDataEnd := tglpHeaderStart + int(tglp.SectionSize)
	expectedTGLP := bffntRaw[tglpHeaderStart:tglpDataEnd]
	assert.Equal(t, len(expectedTGLP), len(encodedTGLP), "TGLP encoding did not produce the correct amount of bytes")

	var cwdhList []CWDH
	cwdhList = DecodeCWDHs(bffntRaw, finf.CWDHOffset)
	encodedCWDHs := EncodeCWDHs(cwdhList, int(finf.CWDHOffset))
	cwdhStart := finf.CWDHOffset - 8
	cwdhEnd := int(cwdhStart) + totalCwdhSectionSize(cwdhList)
	expectedCWDHs := bffntRaw[cwdhStart:cwdhEnd]
	assert.Equal(t, expectedCWDHs, encodedCWDHs, "CWDH encoding did not produce the correct results")

	var cmapList []CMAP
	cmapList = DecodeCMAPs(bffntRaw, finf.CMAPOffset)
	encodedCMAPs := EncodeCMAPs(cmapList, int(finf.CMAPOffset))
	cmapStart := finf.CMAPOffset - 8
	cmapEnd := int(cmapStart) + totalCmapSectionSize(cmapList)
	expectedCMAPs := bffntRaw[cmapStart:cmapEnd]
	assert.Equal(t, expectedCMAPs, encodedCMAPs, "CMAP encoding did not produce the correct results")

	var encodedKRNG []byte
	if strings.Index(string(bffntRaw), KRNG_MAGIC_HEADER) != -1 {
		var krng KRNG
		krng.Decode(bffntRaw)
		krngStart := uint32(strings.Index(string(bffntRaw), KRNG_MAGIC_HEADER))
		encodedKRNG = krng.Encode(krngStart)
		krngEnd := krngStart + krng.SectionSize
		expectedKRNG := bffntRaw[krngStart:krngEnd]
		assert.Equal(t, expectedKRNG, encodedKRNG, "KRNG encoding did not produce the correct results")
	}

	// verify all bytes accounted for
	// totalBytesEncoded := len(encodedFFNT) + len(encodedFINF) + len(encodedTGLP) + len(encodedCWDHs) + len(encodedCMAPs) + len(encodedKRNG)
	// assert.Equal(t, len(bffntRaw), totalBytesEncoded, "the amount of bytes should be the same")

	// verifyUpscale(t, bffntRaw)
	var bffnt BFFNT
	bffnt.Decode(bffntRaw)
	bffnt.Upscale(1)
	encodedUpscaled := bffnt.Encode()
	verifyBffnt(t, encodedUpscaled)

	bffnt.Decode(bffntRaw)
	bffnt.Upscale(2)
	encodedUpscaled = bffnt.Encode()
	verifyBffnt(t, encodedUpscaled)

	bffnt.Decode(bffntRaw)
	bffnt.Upscale(1.1)
	encodedUpscaled = bffnt.Encode()
	verifyBffnt(t, encodedUpscaled)

	bffnt.Decode(bffntRaw)
	bffnt.Upscale(1.2)
	encodedUpscaled = bffnt.Encode()
	verifyBffnt(t, encodedUpscaled)

}

// Sanity checking a bffnt file. Good for verifying the integrity of a bffnt after editing.
func verifyBffnt(t *testing.T, bffntRaw []byte) {
	ffntStart := strings.Index(string(bffntRaw), FFNT_MAGIC_HEADER)
	finfStart := strings.Index(string(bffntRaw), FINF_MAGIC_HEADER)
	tglpStart := strings.Index(string(bffntRaw), TGLP_MAGIC_HEADER)

	cwdhRegex := regexp.MustCompile(CWDH_MAGIC_HEADER)
	cwdhStartList := cwdhRegex.FindAllIndex(bffntRaw, -1) // [][]int, []({start index, end index})
	cwdhStart := cwdhStartList[0][0]

	cmapRegex := regexp.MustCompile(CMAP_MAGIC_HEADER)
	cmapStartList := cmapRegex.FindAllIndex(bffntRaw, -1)
	cmapStart := cmapStartList[0][0]

	krngStart := strings.Index(string(bffntRaw), KRNG_MAGIC_HEADER)

	var ffnt FFNT
	var finf FINF
	var tglp TGLP
	var cwdhList []CWDH
	var cmapList []CMAP
	var krng KRNG

	ffnt.Decode(bffntRaw)
	finf.Decode(bffntRaw)
	tglp.Decode(bffntRaw)
	cwdhList = DecodeCWDHs(bffntRaw, finf.CWDHOffset)
	cmapList = DecodeCMAPs(bffntRaw, finf.CMAPOffset)
	krng.Decode(bffntRaw)

	assertFail(t, 0, ffntStart, "ffnt should start at the byte 0")
	assertFail(t, FFNT_MAGIC_HEADER, ffnt.MagicHeader, `ffnt magic header should be "FFNT"`)
	assertFail(t, FFNT_MAGIC_HEADER, string(bffntRaw[ffntStart:ffntStart+4]), `ffnt magic header should be "FFNT"`)
	assertFail(t, FFNT_HEADER_SIZE, int(ffnt.SectionSize), "ffnt header size should be 20")
	assertFail(t, int(ffnt.TotalFileSize), len(bffntRaw), "ffnt file size should be the bffnt size")

	assertFail(t, 20, finfStart, "finf should start at byte 20")
	assertFail(t, FINF_MAGIC_HEADER, finf.MagicHeader, `finf magic header should be "FINF"`)
	assertFail(t, FINF_MAGIC_HEADER, string(bffntRaw[finfStart:finfStart+4]), `finf magic header should be "FINF"`)
	assertFail(t, FINF_HEADER_SIZE, int(finf.SectionSize), "finf header size should be 32")
	assertFail(t, tglpStart, int(finf.TGLPOffset-8), "finf.TGLPOffset should match the one found by regex")
	assertFail(t, cwdhStart, int(finf.CWDHOffset-8), "finf.CWDHOffset should match the first cwdh offset found by regex")
	assertFail(t, cmapStart, int(finf.CMAPOffset-8), "finf.CMAPOffset should match the first cmap offset found by regex")

	tglpPaddingStart := 84
	tglpPaddingEnd := int(tglp.SheetDataOffset) // exclusive
	tglpPaddingSize := tglpPaddingEnd - tglpPaddingStart
	tglpDataSize := cwdhStart - int(tglp.SheetDataOffset)
	assertFail(t, 52, tglpStart, "tglp should start at byte 52")
	assertFail(t, TGLP_MAGIC_HEADER, tglp.MagicHeader, `tglp magic header should be "TGLP"`)
	assertFail(t, TGLP_MAGIC_HEADER, string(bffntRaw[tglpStart:tglpStart+4]), `tglp magic header should be "TGLP"`)
	assertFail(t, true, allZero(bffntRaw[tglpPaddingStart:tglpPaddingEnd]), "bytes in tglp padding should all be zero'd")
	assertFail(t, cwdhStart-tglpStart, int(tglp.SectionSize), "tglp.SectionSize should match cwdhStart-tglpStart")
	assertFail(t, int(tglp.SectionSize), TGLP_HEADER_SIZE+tglpPaddingSize+tglpDataSize, "all tglp sections added together should equal the Section size")
	assertFail(t, tglpDataSize, int(tglp.SheetSize)*int(tglp.NumOfSheets), "tglp.SheetSize and NumOfSheets should be the same as data size")
	switch tglp.SheetImageFormat {
	case 12:
		// There seems to be a minimum of 65536 (Uint16Max). Ancient_00 observes this.
		sheetArea := math.Max(math.Ceil(float64(tglp.SheetWidth)*float64(tglp.SheetHeight)/float64(2)), 65536)
		assertFail(t, int(tglp.SheetSize), int(sheetArea), "SheetWidth*SheetHeight == SheetSize/2 when ImageFormat is 12 (ETC1)")
	case 8:
		assertFail(t, int(tglp.SheetSize), int(tglp.SheetWidth)*int(tglp.SheetHeight), "SheetWidth*SheetHeight == SheetSize when ImageFormat is 8 (A8)")
	default:
		panic(fmt.Sprintf("SheetWidth, SheetHeight, SheetSize ratio for image format %d not yet coded.", tglp.SheetImageFormat))
	}
	assertFail(t, int(52+tglp.SectionSize), cwdhStart, "cwdh should start whend tglp ends")

	// verify cwdh
	pos := 52 + tglp.SectionSize
	for i, cwdhStartEnd := range cwdhStartList {
		currCWDHStart := cwdhStartEnd[0]
		currCWDH := cwdhList[i]
		assertFail(t, int(pos), currCWDHStart, "cwdh starts when the previous ends")
		assertFail(t, CWDH_MAGIC_HEADER, currCWDH.MagicHeader, `cwdh magic header should be "CWDH"`)
		assertFail(t, CWDH_MAGIC_HEADER, string(bffntRaw[pos:pos+4]), `cwdh magic header should be "CWDH"`)
		assertFail(t, int(currCWDH.EndIndex-currCWDH.StartIndex+1), len(currCWDH.Glyphs), `cwdh did not read in the correct amount of glyphs`)

		cwdhDataStart := int(pos) + CWDH_HEADER_SIZE
		cwdhDataLen := 3 * len(currCWDH.Glyphs)
		cwdhDataEnd := cwdhDataStart + cwdhDataLen
		cwdhPaddingStart := cwdhDataEnd
		cwdhPaddingLen := paddingToNext4ByteBoundary(cwdhPaddingStart)
		cwdhPaddingEnd := cwdhPaddingStart + cwdhPaddingLen // exclusive
		assertFail(t, true, allZero(bffntRaw[cwdhPaddingStart:cwdhPaddingEnd]), "bytes in cwdh padding should all be zero'd")
		assertFail(t, 0, int(cwdhPaddingEnd)%4, "cwdh does not pad to 4 byte boundary")

		if i == (len(cwdhList) - 1) { // last cwdh
			assertFail(t, 0, int(currCWDH.NextCWDHOffset), "The last CWDH's NextCWDHOffset value should be 0 to terminate the list")
			firstCMAPStart := cmapStartList[0][0]
			assertFail(t, firstCMAPStart, int(cwdhPaddingEnd), "last cwdh padding end should == first cmap pad start")
		} else {
			nextCWDHStart := cwdhStartList[i+1][0]
			assertFail(t, nextCWDHStart, int(cwdhPaddingEnd), "current cwdh padding end should == next cwdh pad start")
			assertFail(t, nextCWDHStart, int(currCWDH.NextCWDHOffset-8), "The next CWDH's NextCWDHOffset value should be the same as the one found by regex")
		}
		assertFail(t, int(currCWDH.SectionSize), CWDH_HEADER_SIZE+cwdhDataLen+cwdhPaddingLen, "calculated cwdh size does not match section size")

		pos += cwdhList[i].SectionSize
		assertFail(t, 0, int(pos)%4, "cwdh does not end at 4 byte boundary")
	}

	// verify cmap
	for i, cmapStartEnd := range cmapStartList {
		currCMAPStart := cmapStartEnd[0]
		currCMAP := cmapList[i]
		assertFail(t, int(pos), currCMAPStart, "cmap starts when the previous ends")
		assertFail(t, CMAP_MAGIC_HEADER, currCMAP.MagicHeader, `cmap magic header should be "CMAP"`)
		assertFail(t, CMAP_MAGIC_HEADER, string(bffntRaw[pos:pos+4]), `cmap magic header should be "CMAP"`)
		assertFail(t, len(currCMAP.CharAscii), len(currCMAP.CharIndex), "There should be an equal amount of characters and indexes recorded")

		var cmapDataLen int
		switch currCMAP.MappingMethod {
		case 0:
			cmapDataLen = 2
			assertFail(t, len(currCMAP.CharIndex), int(currCMAP.CodeEnd-currCMAP.CodeBegin+1), "CodeEnd CodeBegin check failed")
		case 1:
			cmapDataLen = 2 * int(currCMAP.CodeEnd-currCMAP.CodeBegin+1)
			assertFail(t, len(currCMAP.CharIndex), int(currCMAP.CodeEnd-currCMAP.CodeBegin+1), "CodeEnd CodeBegin check failed")
		case 2:
			cmapDataLen = 2 + 4*int(currCMAP.CharacterCount)
			assertFail(t, uint16(0), currCMAP.CodeBegin, "Unused codeBegin in scan mapping method (2) set to 0")
			assertFail(t, uint16(65535), currCMAP.CodeEnd, "Unused codeEnd in scan mapping method (2) set to 65535 (uint16Max)")
			assertFail(t, int(currCMAP.CharacterCount), len(currCMAP.CharIndex), "number of character index should equal character count")
		default:
			panic(fmt.Sprintf("unknown mapping method: %d", currCMAP.MappingMethod))
		}

		cmapDataStart := currCMAPStart + CMAP_HEADER_SIZE
		cmapDataEnd := cmapDataStart + cmapDataLen
		cmapPaddingStart := cmapDataEnd
		cmapPaddingLen := paddingToNext4ByteBoundary(cmapPaddingStart)
		cmapPaddingEnd := cmapPaddingStart + cmapPaddingLen // exclusive
		assertFail(t, true, allZero(bffntRaw[cmapPaddingStart:cmapPaddingEnd]), "bytes in cmap padding should all be zero'd")
		assertFail(t, 0, int(cmapPaddingEnd)%4, "cmap does not pad to 4 byte boundary")

		if i == (len(cmapList) - 1) { // last cmap
			assertFail(t, 0, int(currCMAP.NextCMAPOffset), "The last CMAP's NextCMAPOffset value should be 0 to terminate the list")
			// 	firstCMAPStart := cmapStartList[0][0]
			if krngStart != -1 {
				assertFail(t, krngStart, int(cmapPaddingEnd), "last cmap padding end should should be the start of krng found by regex")
			} else {
				assertFail(t, len(bffntRaw), int(cmapPaddingEnd), "last cmap padding end should be the last bffnt byte")
			}
		} else {
			nextCMAPStart := cmapStartList[i+1][0]
			assertFail(t, nextCMAPStart, int(cmapPaddingEnd), "current cmap padding end should == next cmap pad start")
			assertFail(t, nextCMAPStart, int(currCMAP.NextCMAPOffset-8), "The next CMAP's NextCMAPOffset value should be the same as the one found by regex")
		}
		assertFail(t, int(currCMAP.SectionSize), CMAP_HEADER_SIZE+cmapDataLen+cmapPaddingLen, "calculated cmap size does not match section size")

		pos += cmapList[i].SectionSize
		assertFail(t, 0, int(pos)%4, "cmap does not end at 4 byte boundary")
	}

	// verify krng
	if krngStart != -1 {
		assertFail(t, int(pos), krngStart, "krng should start when cmap ends")
		firstCharCount := len(krng.KerningTable)
		var secondCharCount int
		for _, secondCharPairs := range krng.KerningTable {
			secondCharCount += len(secondCharPairs)
		}
		krngDataStart := int(pos) + KRNG_HEADER_SIZE
		krngDataLen := 2 + 4*firstCharCount + 2*firstCharCount + 4*secondCharCount
		krngDataEnd := krngDataStart + krngDataLen
		krngPaddingStart := krngDataEnd
		krngPaddingLen := paddingToNext4ByteBoundary(krngPaddingStart)
		krngPaddingEnd := krngPaddingStart + krngPaddingLen

		assertFail(t, true, allZero(bffntRaw[krngPaddingStart:krngPaddingEnd]), "bytes in krng padding should all be zero'd")
		assertFail(t, int(krng.SectionSize), KRNG_HEADER_SIZE+krngDataLen+krngPaddingLen, "calculated krng size does not match section size")

		pos += krng.SectionSize
		assertFail(t, 0, int(pos)%4, "krng should end on a 4 byte boundary")
	}

	// general checks
	// TODO: verify that there is enough space for all Chars. Useful for rowcount miscalculation during upscale.
	// TODO: verify that there is matching amount of cmap indexes as cwdh attributes

	assertFail(t, int(pos), len(bffntRaw), "Position of our byte counter should be at the end. There are unaccounted bytes at the end")
	assertFail(t, 0, int(pos)%4, "bffnt should end on a 4 byte boundary")
}

// used to check if all padded bytes are zero
func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// File hashing used to verify test file is authentic and original
func hash_file_md5(filePath string) (string, error) {
	// Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	// Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	// Tell the program to call the following function when the current function returns
	defer file.Close()

	// Open a new hash interface to write to
	hash := md5.New()

	// Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	// Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	// Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil

}

func assertFail(t *testing.T, expected interface{}, actual interface{}, errMsg string) {
	if !assert.Equal(t, expected, actual, errMsg) {
		t.FailNow()
	}
}
