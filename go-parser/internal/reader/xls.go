package reader

import (
	"fmt"

	"github.com/pbnjay/grate"
	// Registers the BIFF backend with grate.Open.
	_ "github.com/pbnjay/grate/xls"
)

// xlsWorkbook reads legacy BIFF through pbnjay/grate.
//
// grate replaced extrame/xls, which was measured against calamine ground truth
// and found to corrupt 69% of cells and drop a further 28%: undecoded UTF-16LE
// and BIFF record framing leaked into cell values, content moved between rows
// and columns, and tail rows came back blank. That was charset-independent and
// unfixable from the outside. grate reproduces calamine exactly on the same
// files.
//
// The one normalisation grate needs is trailing blank rows: it yields rows past
// the end of calamine's used range (one for a populated sheet, two for an empty
// one), so trailing all-blank rows are trimmed. Note this is the opposite of
// the xlsx path, where excelize already trims and calamine keeps a 1x1 empty
// range — in both cases the rule is "match calamine's used range".
type xlsWorkbook struct {
	sheets []Sheet
	rows   [][][]Cell
}

func openXLS(path string) (Workbook, error) {
	wb, err := grate.Open(path)
	if err != nil {
		return nil, fmt.Errorf("grate open %s: %w", path, err)
	}
	defer wb.Close()

	names, err := wb.List()
	if err != nil {
		return nil, fmt.Errorf("grate list %s: %w", path, err)
	}

	out := &xlsWorkbook{}
	for idx, name := range names {
		sh, err := wb.Get(name)
		if err != nil {
			return nil, fmt.Errorf("grate get %s/%s: %w", path, name, err)
		}

		var raw [][]string
		for sh.Next() {
			row := sh.Strings()
			cp := make([]string, len(row))
			for i, v := range row {
				cp[i] = demergeMarker(v)
			}
			raw = append(raw, cp)
		}

		// Trim to calamine's used range.
		height := len(raw)
		for height > 0 && rowAllBlank(raw[height-1]) {
			height--
		}
		raw = raw[:height]

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
				row[j] = Cell{Str: v, IsEmpty: v == ""}
			}
			cells[i] = padRow(row, width)
		}

		out.sheets = append(out.sheets, Sheet{Index: idx, Name: name, Height: height, Width: width})
		out.rows = append(out.rows, cells)
	}
	return out, nil
}

// demergeMarker blanks grate's merged-cell continuation markers.
//
// grate fills the cells covered by a merge with sentinel runes; calamine
// reports them as empty. In this corpus they occur only in the merged title
// block of the 2016 spreadsheets (19 cells in rows 0-2 of one file). Only an
// exact whole-value match is blanked, so a real cell that merely contains an
// arrow is untouched.
func demergeMarker(v string) string {
	switch v {
	case grate.ContinueColumnMerged, grate.EndColumnMerged,
		grate.ContinueRowMerged, grate.EndRowMerged:
		return ""
	}
	return v
}

func (w *xlsWorkbook) Sheets() []Sheet { return w.sheets }

func (w *xlsWorkbook) EachRow(sheetIdx int, fn RowFunc) error {
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

func (w *xlsWorkbook) Close() error { return nil }
