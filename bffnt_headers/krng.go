package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"sort"
	"strings"
)

type kerningPair struct {
	SecondChar   uint16
	KerningValue int16 // kerning can be negative
}

type KRNG struct { // Offset  Size  Description
	MagicHeader string // 0x00    0x04  Magic Header (KRNG)
	SectionSize uint32 // 0x04    0x04  Section Size
	// FirstCharCount     0x08    0x02  Amount of First Chars

	// FirstChar          0x0A    0x02  First char in a pair
	// OffsetToPairArray  0x0C    0x02  Offset to the array of second characters divided by 2

	// kerning pair array
	// PairCount          0x0E    0x02  Amount of kerningPairs (second character, kerning value)
	// SecondCharacter    0x10    0x02  Second character in a pair
	// KerningValue       0x12    0x02  Kerning value

	KerningTable map[uint16][]kerningPair
	// Key = First character of a pair
	// In order to save space, Nintendo represents the kerning pairs as a map of
	// pair arrays. The key of the map is the first character of the pair. The
	// pair is made up of the second character and the kerning value.
	// visual example:
	//
	// First Character
	//  |        +-------SecondChar
	//  |        |    +--------------Kerning value
	//  |        |    |
	//  V        V    V
	// [ A ] | [( V, -1 ), ( W, -1 ), ( Y, -1 )]
	// [ L ] | [( V, -1 ), ( T, -1 ), ( W, -1 )]
	// [ P ] | [( d, -2 ), ( g, -2 ), ( y, -1 )]
}

// The kerning index table doesn't seem to be recorded in any headers. It is
// most likely usually the last section.
func (krng *KRNG) Decode(bffntRaw []byte) {
	// Since the kerning offset is not recorded we need to find it first.
	headerStart := strings.Index(string(bffntRaw), KRNG_MAGIC_HEADER)
	if headerStart == -1 {
		return
	}

	headerEnd := headerStart + KRNG_HEADER_SIZE
	headerRaw := bffntRaw[headerStart:headerEnd]
	assertEqual(KRNG_HEADER_SIZE, len(headerRaw))

	krng.MagicHeader = string(headerRaw[0:4])
	krng.SectionSize = binary.BigEndian.Uint32(headerRaw[4:8])

	// fmt.Println(krng.MagicHeader)
	// fmt.Println(krng.SectionSize)

	dataEnd := headerStart + int(krng.SectionSize)
	data := bffntRaw[headerEnd:dataEnd]

	// fmt.Println(dataEnd - headerStart)

	// The first two bytes are the amount of FirstChars
	firstCharCount := binary.BigEndian.Uint16(data[0:2])
	dataPos := 2

	kerningMap := make(map[uint16][]kerningPair, 0)
	// loop through first chars and their offset to the array of kerning pairs
	for i := 0; i < int(firstCharCount); i++ {
		firstChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		secondCharOffset := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
		dataPos += 4

		// fmt.Println("============================")
		// fmt.Println("FirstChar:", string(firstChar))
		// fmt.Println("SecondChar offset:", secondCharOffset)

		// The real offset must be multiplied by 2. This might be the case
		// because a single uint16 might not be big enough for an offset if the
		// kerning table is too large
		secondCharOffset = secondCharOffset * 2
		secondCharCount := binary.BigEndian.Uint16(data[secondCharOffset : secondCharOffset+2])

		pairDataStart := secondCharOffset + 2
		pairDataEnd := secondCharOffset + 2 + secondCharCount*4
		pairData := data[pairDataStart:pairDataEnd]

		// fmt.Println("SecondChar real offset:", secondCharOffset)
		// fmt.Println("SecondChar Count:", secondCharCount)

		// Go to offset and record kerning pairs for this char
		pairPos := 0
		kerningPairSlice := make([]kerningPair, 0)
		for j := 0; j < int(secondCharCount); j++ {
			secondChar := binary.BigEndian.Uint16(pairData[pairPos : pairPos+2])
			kerningValue := int16(binary.BigEndian.Uint16(pairData[pairPos+2 : pairPos+4]))

			kerningPairSlice = append(kerningPairSlice, kerningPair{secondChar, kerningValue})

			pairPos += 4
		}

		kerningMap[firstChar] = kerningPairSlice

		// fmt.Println(string(firstChar), kerningMap[firstChar])
	}

	// find where the last table data resides so we can verify leftover bytes
	// assume that the last

	// leftover bytes?
	// leftover := uint16(binary.BigEndian.Uint16(data[dataPos : dataPos+2]))
	// fmt.Println(leftover)
}

func (krng *KRNG) Encode() []byte {
	if len(krng.KerningTable) == 0 {
		return []byte{}
	}

	var dataBuf bytes.Buffer
	dataWriter := bufio.NewWriter(&dataBuf)

	FirstChars := getFirstCharsOrdered(krng.KerningTable)

	// Nintendo divides the actual second character data offset by 2 before
	// recording it. This is because the kerning table consist of only uint16s
	// and int16s which means bytes are written in pairs (2 bytes).  By
	// exploiting the fact that the second character data offset is guaranteed
	// to be an even number, by halving the recorded offset, the theoretical
	// maximum size of the kerning table is increased by a factor of 2x.
	secondCharDataOffset := len(FirstChars) * 2

	// Write first chars
	for _, firstChar := range FirstChars {
		binaryWrite(dataWriter, firstChar)
		binaryWrite(dataWriter, uint16(secondCharDataOffset/2))

		secondCharDataOffset += 2 // 2 bytes for second char count
		secondCharDataOffset += 2 * len(krng.KerningTable[firstChar])
	}

	// Write kerning Data
	for _, firstChar := range FirstChars {
		secondCharCount := uint16(len(krng.KerningTable[firstChar]))
		binaryWrite(dataWriter, secondCharCount)

		for _, kerningPair := range krng.KerningTable[firstChar] {
			binaryWrite(dataWriter, kerningPair.SecondChar)
			binaryWrite(dataWriter, kerningPair.KerningValue)
		}
	}
	dataWriter.Flush()
	krngData := dataBuf.Bytes()

	krng.SectionSize = uint32(KRNG_HEADER_SIZE + len(krngData))

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	// Write raw data of the header and data
	_, _ = w.Write([]byte(KRNG_MAGIC_HEADER))
	binaryWrite(w, krng.SectionSize)
	_, _ = w.Write(krngData)

	w.Flush()
	return buf.Bytes()
}

// takes the kerning table and returns the inputs in order.  Not functionally
// needed. But easier to read when in order.
func getFirstCharsOrdered(kerningTable map[uint16][]kerningPair) []uint16 {
	res := make([]uint16, len(kerningTable))

	orderedFirstChars := make([]int, len(kerningTable))
	for char, _ := range kerningTable {
		orderedFirstChars = append(orderedFirstChars, int(char))
	}
	sort.Ints(orderedFirstChars)

	// convert back into uint16
	for i, char := range orderedFirstChars {
		res[i] = uint16(char)
	}

	return res
}

func (krng *KRNG) Upscale(scale uint8) {
	for _, kPairs := range krng.KerningTable {
		for _, pair := range kPairs {
			pair.KerningValue *= int16(scale)
		}
	}
}
