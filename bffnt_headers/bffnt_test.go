package bffnt_headers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBFFNT(t *testing.T) {
	testCase(t, "../WiiU_fonts/botw/Ancient/Ancient_00.bffnt", "bc6525a0089b9ddc90a2f25a1d68291e")
	testCase(t, "../WiiU_fonts/botw/Special/Special_00.bffnt", "4d973f84b287d787e5b1ed8d1fd82799")
	testCase(t, "../WiiU_fonts/botw/Caption/Caption_00.bffnt", "efc0070d11289b18f28525a755e75acb")
	testCase(t, "../WiiU_fonts/botw/Normal/Normal_00.bffnt", "8d7f1ec5872da263a95a5937ccd8a372")
	testCase(t, "../WiiU_fonts/botw/NormalS/NormalS_00.bffnt", "f993a5822f3ce05e51e0440b46bd1345")
	testCase(t, "../WiiU_fonts/botw/NormalS/NormalS_00.bffnt", "f993a5822f3ce05e51e0440b46bd1345")

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
	// encodedTGLP := tglp.Encode()

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

}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func hash_file_md5(filePath string) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	//Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	//Tell the program to call the following function when the current function returns
	defer file.Close()

	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil

}
