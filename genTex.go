package main

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	const (
		scale = 3
		// attributes of NormalS_00.bffnt
		cellWidth   = 24 * scale
		cellHeight  = 30 * scale
		columnCount = 20
		rowCount    = 33

		baseLine    = 23 * scale // ascent
		sheetHeight = 1024 * scale
		sheetWidth  = 512 * scale

		realCellWidth  = cellWidth + 1
		realCellHeight = cellHeight + 1
	)

}
