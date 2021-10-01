package bffnt_headers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
)

type CFNT struct { //       Offset  Size  Description
	MagicHeader   string // 0x00    0x04  Magic Header (either CFNT or CFNU or FFNT)
	Endianness    uint16 // 0x04    0x02  Endianness (0xFEFF = little, 0xFFFE = big)
	SectionSize   uint16 // 0x06    0x02  Header Size
	Version       uint32 // 0x08    0x04  Version (observed to be 0x03000000)
	TotalFileSize uint32 // 0x0C    0x04  File size (the total)
	BlockReadNum  uint32 // 0x10    0x04  Number of "blocks" to read

	// It looks like BlockReadNum is always some multiple of 2^16 (65536 in
	// decimal. 0x10000 in HEX). Unclear wether this can break a font. It might
	// be that its a suggestion to the system to it can block read at a time.
	// perhaps it is ok to change this number around. Change this bit and see if botw crashes.

	// remainder := (cfnt.TotalFileSize % 65536)
	// quotient := (cfnt.TotalFileSize - remainder) / 65536
	// calculatedBlockReadNum := int((quotient + 1) * 65536)
	// fmt.Println(tglp.SheetSize)
	// fmt.Println(remainder)
	// fmt.Println(quotient)
	// fmt.Println(calculatedBlockReadNum)
	// assertEqual(calculatedBlockReadNum, int(cfnt.BlockReadNum))
}

func (cfnt *CFNT) Decode(raw []byte) {
	headerStart := 0
	headerEnd := headerStart + CFNT_HEADER_SIZE
	headerRaw := raw[headerStart:headerEnd]
	assertEqual(CFNT_HEADER_SIZE, len(headerRaw))

	cfnt.MagicHeader = string(headerRaw[0:4])
	cfnt.Endianness = binary.BigEndian.Uint16(headerRaw[4:6])
	cfnt.SectionSize = binary.BigEndian.Uint16(headerRaw[6:8])
	cfnt.Version = binary.BigEndian.Uint32(headerRaw[8:12])
	cfnt.TotalFileSize = binary.BigEndian.Uint32(headerRaw[12:16])
	cfnt.BlockReadNum = binary.BigEndian.Uint32(headerRaw[16:CFNT_HEADER_SIZE])

	if Debug {
		pprint(cfnt)
		fmt.Printf("Read section total of %d bytes\n", headerEnd-headerStart)
		fmt.Println("Byte offsets start(inclusive) to end(exclusive)================")
		fmt.Printf("header %d(inclusive) to %d(exclusive)\n", headerStart, headerEnd)
		fmt.Println()
	}
}

func (cfnt *CFNT) Encode() []byte {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	_, _ = w.Write([]byte(cfnt.MagicHeader))
	_ = binary.Write(w, binary.BigEndian, cfnt.Endianness)
	_ = binary.Write(w, binary.BigEndian, cfnt.SectionSize)
	_ = binary.Write(w, binary.BigEndian, cfnt.Version)
	_ = binary.Write(w, binary.BigEndian, cfnt.TotalFileSize)
	_ = binary.Write(w, binary.BigEndian, cfnt.BlockReadNum)
	w.Flush()

	assertEqual(CFNT_HEADER_SIZE, len(buf.Bytes()))
	return buf.Bytes()
}
