// Package reader wraps the two spreadsheet libraries behind one contract.
//
// Cell rendering and sheet geometry are pinned by the frozen fidelity oracle in
// parser/testdata/reader-fidelity-hashes.tsv. "The reference" below means that
// pinned rendering: whenever an underlying library disagrees with it, this
// package corrects the library, never the oracle.
//
// The *contract* is row-at-a-time: callers receive one row per callback and
// never hold the workbook. The two implementations do not honour that in
// memory — both decode the whole workbook into [][][]Cell up front, because
// neither underlying library exposes a usable row cursor. data/2017/ha-noi.xls
// alone holds 72k rows across two sheets, so a caller that assumed streaming
// memory here would be wrong.
package reader

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Cell is one spreadsheet cell rendered as reference text.
//
// IsEmpty marks a cell with no value, as opposed to a string cell that happens
// to be empty. Both render "" and both count as blank in IsAllBlank and in
// transform, so the flag is diagnostic only — never compare on it.
type Cell struct {
	Str     string
	IsEmpty bool
}

// Sheet carries a sheet's identity and geometry. Height and Width describe the
// reference used range; every used range in the corpus starts at (0,0),
// verified across every input file.
type Sheet struct {
	Index  int
	Name   string
	Height int
	Width  int
}

// RowFunc receives each row of each sheet. Rows are padded to the sheet's used
// width — width is load-bearing because every column read downstream is
// positional and falls back to "" when the index is out of range, so a short
// row silently NULLs its tail columns.
type RowFunc func(sheet Sheet, rowIdx int, row []Cell) error

// Workbook is one opened spreadsheet.
type Workbook interface {
	// Sheets returns sheet identity and geometry in workbook order.
	Sheets() []Sheet
	// EachRow streams every row of the given sheet in order.
	EachRow(sheetIdx int, fn RowFunc) error
	Close() error
}

// Open dispatches on file extension.
func Open(path string) (Workbook, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xls":
		return openXLS(path)
	case ".xlsx", ".xlsm":
		return openXLSX(path)
	default:
		return nil, fmt.Errorf("unsupported extension: %s", path)
	}
}

// padRow extends row to width with empty cells, and truncates if longer.
func padRow(row []Cell, width int) []Cell {
	if len(row) == width {
		return row
	}
	if len(row) > width {
		return row[:width]
	}
	out := make([]Cell, width)
	copy(out, row)
	for i := len(row); i < width; i++ {
		out[i] = Cell{IsEmpty: true}
	}
	return out
}
