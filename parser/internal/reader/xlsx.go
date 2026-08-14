package reader

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// xlsxWorkbook reads OOXML through excelize.
//
// Two excelize behaviours must be corrected to match the reference:
//
//  1. GetRows applies the cell number format by default, while the reference
//     renders the underlying value. RawCellValue: true disables that.
//  2. GetRows trims trailing blank cells, so rows are ragged; the reference used
//     range is rectangular. Rows are padded back out to the sheet width.
type xlsxWorkbook struct {
	f      *excelize.File
	sheets []Sheet
	rows   [][][]Cell // [sheetIdx][rowIdx][colIdx]
}

func openXLSX(path string) (Workbook, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("excelize open %s: %w", path, err)
	}

	wb := &xlsxWorkbook{f: f}
	crFixups := buildCRFixups(path)
	for idx, name := range f.GetSheetList() {
		raw, err := f.GetRows(name, excelize.Options{RawCellValue: true})
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("excelize GetRows %s/%s: %w", path, name, err)
		}

		// Do NOT trim trailing blank rows: the reference used range keeps them,
		// and excelize's GetRows already drops trailing fully-empty rows itself.
		//
		// One correction is needed. A sheet holding a single empty
		// shared-string cell at A1 is a 1x1 range to the reference, while
		// GetRows returns nothing for it. A genuinely empty sheet (230 of them
		// in 2016) is height 0 either way.
		// GetCellType tells the two apart: the empty-shared-string cell exists
		// in the XML and types as CellTypeSharedString, an absent cell types as
		// CellTypeUnset.
		if len(raw) == 0 {
			if t, terr := f.GetCellType(name, "A1"); terr == nil && t != excelize.CellTypeUnset {
				raw = [][]string{{""}}
			}
		}
		height := len(raw)

		width := 0
		for _, r := range raw {
			if len(r) > width {
				width = len(r)
			}
		}

		cells := make([][]Cell, len(raw))
		for i, r := range raw {
			row := make([]Cell, len(r))
			for j, v := range r {
				if fixed, ok := crFixups[v]; ok {
					v = fixed
				} else {
					v = normalizeNumeric(f, name, j, i, v)
				}
				row[j] = Cell{Str: v, IsEmpty: v == ""}
			}
			cells[i] = padRow(row, width)
		}

		wb.sheets = append(wb.sheets, Sheet{Index: idx, Name: name, Height: height, Width: width})
		wb.rows = append(wb.rows, cells)
	}
	return wb, nil
}

func rowAllBlank(r []string) bool {
	for _, v := range r {
		if v != "" {
			return false
		}
	}
	return true
}

func (w *xlsxWorkbook) Sheets() []Sheet { return w.sheets }

func (w *xlsxWorkbook) EachRow(sheetIdx int, fn RowFunc) error {
	if sheetIdx < 0 || sheetIdx >= len(w.sheets) {
		return fmt.Errorf("sheet index %d out of range", sheetIdx)
	}
	sh := w.sheets[sheetIdx]
	for i, row := range w.rows[sheetIdx] {
		if err := fn(sh, i, row); err != nil {
			return err
		}
	}
	return nil
}

func (w *xlsxWorkbook) Close() error { return w.f.Close() }
