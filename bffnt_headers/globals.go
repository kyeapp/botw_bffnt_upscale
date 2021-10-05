package bffnt_headers

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

var (
	Debug bool
)

const (
	// number of bytes for each header size
	CFNT_HEADER_SIZE = 20
	FINF_HEADER_SIZE = 32
	TGLP_HEADER_SIZE = 32
	CWDH_HEADER_SIZE = 16
	CMAP_HEADER_SIZE = 20
	KRNG_HEADER_SIZE = 8

	CFNT_MAGIC_HEADER = "FFNT"
	FINF_MAGIC_HEADER = "FINF"
	TGLP_MAGIC_HEADER = "TGLP"
	CWDH_MAGIC_HEADER = "CWDH"
	CMAP_MAGIC_HEADER = "CMAP"
	KRNG_MAGIC_HEADER = "KRNG"
)

func assertEqual(expected int, actual int) {
	if expected != actual {
		err := fmt.Errorf("%d(actual) does not equal %d(expected)\n", actual, expected)
		handleErr(err)
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Just a wrapper around binary.Write
func binaryWrite(w *bufio.Writer, data interface{}) {
	err := binary.Write(w, binary.BigEndian, data)
	handleErr(err)

	// just call every time. its easy to forget and end up with missing bytes
	w.Flush()
}

func pprint(s interface{}) {
	jsonBytes, err := json.MarshalIndent(s, "", "  ")
	// jsonBytes, err := json.Marshal(s)
	handleErr(err)

	fmt.Printf("%s\n", string(jsonBytes))
}

// It looks like in some cases there can be left over bytes from a section
// after decoding is done. Not a significant amount. Usually 2, 4, or 6 bytes.
// If these bytes are really unused we should expect them to be zero'd out.
func verifyLeftoverBytes(leftovers []byte) {
	if len(leftovers) > 0 {
		fmt.Printf("There are %d bytes left over", len(leftovers))

		for _, singleByte := range leftovers {
			if singleByte != 0 {
				fmt.Println("left over bytes:", leftovers)
				err := fmt.Errorf("There are left over bytes that are not zero'd")
				handleErr(err)
			}
		}
	}
}
