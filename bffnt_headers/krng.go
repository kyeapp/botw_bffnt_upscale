package bffnt_headers

import (
	"encoding/binary"
	"fmt"
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

func (krng *KRNG) Decode(bffntRaw []byte) {
	return

	pos := 536080 // KRNG start
	data := bffntRaw[pos:]

	dataPos := 0
	fmt.Println(string(data[0:4]))
	fmt.Printf("section size: %v\n", binary.BigEndian.Uint32(data[4:8]))
	firstCharCount := binary.BigEndian.Uint16(data[8:10])
	fmt.Printf("amount of FirstChars: %v\n", firstCharCount)

	dataPos += 10

	e := int(firstCharCount)
	for i := 0; i < e; i++ {
		firstChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		offset := binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4])
		dataPos += 4

		fmt.Printf("( '%s', %d )\n", string(firstChar), offset)
	}
	// dataPos is 378

	fmt.Println("SECOND CHARS============================")
	// decode 2nd char?
	for i := 0; i < e; i++ {
		fmt.Println("data index?:", (dataPos-8)/2)
		secondCharCount := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
		dataPos += 2
		fmt.Printf("amount of SecondChars: %v\n", secondCharCount)
		for i := 0; i < int(secondCharCount); i++ {
			secondChar := binary.BigEndian.Uint16(data[dataPos : dataPos+2])
			offset := int16(binary.BigEndian.Uint16(data[dataPos+2 : dataPos+4]))
			dataPos += 4

			fmt.Printf("( '%s', %d )\n", string(secondChar), offset)
		}
	}

	// leftover bytes?
	leftover := uint16(binary.BigEndian.Uint16(data[dataPos : dataPos+2]))
	fmt.Println(leftover)
}

func (krng *KRNG) Encode(bffntRaw []byte) []byte {
	pos := 536080 // KRNG start
	data := bffntRaw[pos:]

	return data
}
