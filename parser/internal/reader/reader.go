// Package reader wraps the two spreadsheet libraries behind one streaming
// contract, mirroring parser/src/reader.rs (which wraps calamine's
// open_workbook_auto for both formats).
//
// The contract is deliberately row-streaming rather than whole-workbook
// materialising: data/2017/ha-noi.xls alone holds 72k rows across two sheets,
// and the Rust original keeps at most one sheet range live at a time.
package reader

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Cell is one spreadsheet cell rendered the way calamine's Data::to_string()
// renders it.
//
// IsEmpty tracks calamine's Data::Empty variant separately from a string cell
// that happens to be empty. Rust distinguishes them at reader.rs:42, but both
// render "" and both count as blank in is_all_blank and in transform, so the
// flag is diagnostic only — never compare on it.
type Cell struct {
	Str     string
	IsEmpty bool
}

// Sheet carries a sheet's identity and geometry. Height and Width describe the
// used range, matching calamine's Range::height()/width(); every used range in
// the corpus starts at (0,0), verified across all 299 files.
type Sheet struct {
	Index  int
	Name   string
	Height int
	Width  int
}

// RowFunc receives each row of each sheet. Rows are padded to the sheet's used
// width — width is load-bearing because every column read downstream is
// positional with an unwrap_or_default() equivalent, so a short row silently
// NULLs its tail columns.
type RowFunc func(sheet Sheet, rowIdx int, row []Cell) error

// Workbook is one opened spreadsheet.
type Workbook interface {
	// Sheets returns sheet identity and geometry in workbook order.
	Sheets() []Sheet
	// EachRow streams every row of the given sheet in order.
	EachRow(sheetIdx int, fn RowFunc) error
	Close() error
}

// Open dispatches on file extension, mirroring calamine's open_workbook_auto.
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
