package bffnt_headers

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBFFNT(t *testing.T) {
	bffntRaw, err := ioutil.ReadFile("../WiiU_fonts/botw/NormalS/NormalS_00.bffnt")
	handleErr(err)

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

	// verify tglp data is good

	// b.CWDHs = bffnt_headers.DecodeCWDHs(bffntRaw, b.FINF.CWDHOffset)
	// b.CMAPs = bffnt_headers.DecodeCMAPs(bffntRaw, b.FINF.CMAPOffset)
	// b.KRNG.Decode(bffntRaw)

	// ffntRaw :=
	// 	finfRaw
	// tglpRaw

	// cwdhsRaw
	// cmapsRaw
	// krngRaw
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
