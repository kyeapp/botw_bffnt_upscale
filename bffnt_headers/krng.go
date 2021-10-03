package bffnt_headers

import (
	"encoding/binary"
)

type kerningPair struct {
	// FirstChar uint16
	SecondChar   uint16
	kerningValue uint16
}

type KRNG struct { // Offset  Size  Description
	MagicHeader string // 0x00    0x04  Magic Header (KRNG)
	SectionSize uint32 // 0x04    0x04  Section Size
	// FirstCharCount     0x08    0x02  amount of First Chars

	// FirstChar          0x0A    0x02  First char in a pair
	// OffsetToPairArray  0x0C    0x02  Offset to the array of second characters (must multiply by 2)

	// kerning pair array
	// pairCount          0x0E    0x02  amount of kerningPairs (second character, kerning value)
	//                    0x10    0x02  second char in a pair
	//                    0x12    0x02  kerning value

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

// THIS IS UNTESTED
func (krng *KRNG) Decode(bffntRaw []byte, startingOffset int) {

	return
	// headerStart := 536080
	headerStart := startingOffset
	headerEnd := headerStart + KRNG_HEADER_SIZE
	headerRaw := bffntRaw[headerStart:headerEnd]
	assertEqual(KRNG_HEADER_SIZE, len(headerRaw))

	krng.MagicHeader = string(headerRaw[0:4])
	krng.SectionSize = binary.BigEndian.Uint32(headerRaw[4:8])

	dataEnd := headerStart + int(krng.SectionSize)
	data := bffntRaw[headerEnd:dataEnd]

	// The first two bytes are the amount of FirstChars
	firstCharCount := binary.BigEndian.Uint16(data[0:2])
	dataPos := 2

	kerningMap := make(map[uint16][]kerningPair, 0)
	// loop through first chars and their offset to the array of kerning pairs
	for i := 0; i < int(firstCharCount); i++ {
		firstChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		secondCharOffset := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
		dataPos += 4

		// fmt.Printf("( '%s', %d )\n", string(firstChar), offset)
		// The real offset must be multiplied by 2. This might be the case
		// because a single uint16 might not be big enough for an offset if the
		// kerning table is too large
		secondCharOffset = secondCharOffset*2 + 8
		secondCharCount := binary.BigEndian.Uint16(data[secondCharOffset : secondCharOffset+2])

		pairData := data[secondCharOffset+2 : secondCharOffset*4]

		// Go to offset and record kerning pairs for this char
		pairPos := 0
		kerningPairSlice := make([]kerningPair, 0)
		for j := 0; j < int(secondCharCount); j++ {
			secondChar := binary.BigEndian.Uint16(pairData[pairPos : pairPos+2])
			kerningValue := uint16(binary.BigEndian.Uint16(pairData[pairPos+2 : pairPos+4]))

			kerningPairSlice = append(kerningPairSlice, kerningPair{secondChar, kerningValue})

			pairPos += 4
		}

		kerningMap[firstChar] = kerningPairSlice
	}

	// leftover bytes?
	// leftover := uint16(binary.BigEndian.Uint16(data[dataPos : dataPos+2]))
	// fmt.Println(leftover)
}

func (krng *KRNG) Encode(bffntRaw []byte) []byte {
	pos := 536080 // KRNG start
	data := bffntRaw[pos:]

	return data
}
