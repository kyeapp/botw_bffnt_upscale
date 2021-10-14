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
	testCase(t, "../WiiU_fonts/botw/NormalS/NormalS_00.bffnt", "f993a5822f3ce05e51e0440b46bd1345")
	testCase(t, "../WiiU_fonts/botw/Normal/Normal_00.bffnt", "8d7f1ec5872da263a95a5937ccd8a372")
	testCase(t, "../WiiU_fonts/botw/Caption/Caption_00.bffnt", "efc0070d11289b18f28525a755e75acb")
}

func testCase(t *testing.T, bffntFile string, expectedFileHash string) {
	t.Log(fmt.Sprintf("Testing %s", bffntFile))
	bffntRaw, err := ioutil.ReadFile(bffntFile)
	handleErr(err)

	//TODO verify the file with an MD5 hash
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
	// TODO: verify tglp data is good

	var cwdhList []CWDH
	cwdhList = DecodeCWDHs(bffntRaw, finf.CWDHOffset)
	encodedCWDH := EncodeCWDHs(cwdhList, int(finf.CWDHOffset))
	cwdhStart := finf.CWDHOffset - 8
	cwdhEnd := int(cwdhStart) + totalCwdhSectionSize(cwdhList)
	expectedCWDH := bffntRaw[cwdhStart:cwdhEnd]
	assert.Equal(t, expectedCWDH, encodedCWDH, "CWDH encoding did not produce the correct results")

	var cmapList []CMAP
	cmapList = DecodeCMAPs(bffntRaw, finf.CMAPOffset)
	encodedCMAP := EncodeCMAPs(cmapList, int(finf.CMAPOffset))
	cmapStart := finf.CMAPOffset - 8
	cmapEnd := int(cmapStart) + totalCmapSectionSize(cmapList)
	expectedCMAP := bffntRaw[cmapStart:cmapEnd]
	assert.Equal(t, expectedCMAP, encodedCMAP, "CMAP encoding did not produce the correct results")

	var krng KRNG
	krng.Decode(bffntRaw)
	krngStart := uint32(strings.Index(string(bffntRaw), KRNG_MAGIC_HEADER))
	encodedKRNG := krng.Encode(krngStart)
	krngEnd := krngStart + krng.SectionSize
	expectedKRNG := bffntRaw[krngStart:krngEnd]
	assert.Equal(t, expectedKRNG, encodedKRNG, "KRNG encoding did not produce the correct results")

	// verify all bytes accounted for
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
