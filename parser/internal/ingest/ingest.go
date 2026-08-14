// Package ingest owns all dataset policy: which sheets to read, which rows are
// headers, which are blank, and how rows are counted.
//
// The reader deliberately has none of this — it reports every sheet and every
// row verbatim, which is what keeps its fidelity independently testable.
package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tiennm99/thptqg/parser/internal/config"
	"github.com/tiennm99/thptqg/parser/internal/reader"
	"github.com/tiennm99/thptqg/parser/internal/transform"
	"github.com/tiennm99/thptqg/parser/internal/writer"
)

// IsHeaderRow reports whether row is a header, by matching its uppercased first
// cell against the configured tokens.
//
// Rows shorter than 3 cells are never headers — a 1- or 2-cell row is a stray
// fragment, not a real header.
func IsHeaderRow(row []reader.Cell, tokens []string) bool {
	if len(row) < 3 {
		return false
	}
	first := strings.ToUpper(strings.TrimSpace(row[0].Str))
	for _, t := range tokens {
		if strings.ToUpper(t) == first {
			return true
		}
	}
	return false
}

// IsAllBlank reports whether every cell is empty or whitespace-only.
//
// Compares on Str only. Cell.IsEmpty is diagnostic: an absent cell and an empty
// string cell both render "" and both count as blank here, so branching on the
// flag would invent a distinction nothing downstream acts on.
func IsAllBlank(row []reader.Cell) bool {
	for _, c := range row {
		if strings.TrimSpace(c.Str) != "" {
			return false
		}
	}
	return true
}

// InputFiles lists a dataset directory's spreadsheets, bytewise-sorted on the
// full path.
//
// The sort is load-bearing, not cosmetic: INSERT OR REPLACE is last-wins, so
// file order decides which row survives a duplicate SBD.
func InputFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read input dir %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".xls", ".xlsx":
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

// DatasetLabel derives the build-log label from the input directory basename.
func DatasetLabel(inputDir string) string {
	base := filepath.Base(strings.TrimRight(inputDir, string(filepath.Separator)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "data"
	}
	return base
}

// RowFn consumes one data row of one sheet, after header skipping.
type RowFn func(sheetIdx int, row []reader.Cell)

// ProcessFile applies sheet selection and per-sheet header skipping, invoking fn
// for every remaining row.
//
// The header check is per SHEET, not per file: firstRow resets inside the sheet
// loop, so a workbook whose second sheet repeats the header has it skipped
// there too.
func ProcessFile(path string, cfg *config.DatasetConfig, fn RowFn) error {
	wb, err := reader.Open(path)
	if err != nil {
		return err
	}
	defer wb.Close()

	sheets := wb.Sheets()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets in %s", path)
	}
	if cfg.Reader.SheetMode == config.SheetModeFirst {
		sheets = sheets[:1]
	}

	for _, sh := range sheets {
		firstRow := true
		err := wb.EachRow(sh.Index, func(s reader.Sheet, _ int, row []reader.Cell) error {
			if firstRow {
				firstRow = false
				if IsHeaderRow(row, cfg.Header.Tokens) {
					return nil
				}
			}
			fn(s.Index, row)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Standard runs the fixed-column path for the 2017-family datasets.
func Standard(cfg *config.DatasetConfig, inputDir, outputPath string) error {
	files, err := InputFiles(inputDir)
	if err != nil {
		return err
	}
	label := DatasetLabel(inputDir)
	fmt.Printf("[build] %s/ → %s  (%d files)\n", label, outputPath, len(files))

	db, err := writer.OpenDB(outputPath)
	if err != nil {
		return err
	}
	defer db.Close()

	stripBlank := cfg.Reader.StripBlankRows

	// One transaction spans the whole dataset directory.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	ins, err := writer.Prepare(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	var st writer.Stats
	for _, file := range files {
		base := filepath.Base(file)
		var fileRows, fileSkipped, fileErrors uint64

		procErr := ProcessFile(file, cfg, func(_ int, row []reader.Cell) {
			allBlank := IsAllBlank(row)
			// Blank rows drop out BEFORE the source-row counter, so they never
			// reach the source total.
			if stripBlank && allBlank {
				return
			}
			st.SourceRows++

			hoTen, soBaoDanh := "", ""
			if cols := cfg.Columns; cols != nil {
				hoTen = cellAt(row, cols.HoTen)
				soBaoDanh = cellAt(row, cols.SoBaoDanh)
			}

			if transform.ValidateRow(hoTen, soBaoDanh, &cfg.Validation, stripBlank, allBlank) != transform.SkipNone {
				fileSkipped++
				return
			}

			parsed, err := transform.TransformRow(row, cfg)
			if err != nil {
				fileErrors++
				return
			}
			if err := ins.Insert(parsed); err != nil {
				fileErrors++
				// Only the first five insert warnings print.
				if st.Errors+fileErrors <= 5 {
					fmt.Fprintf(os.Stderr, "  [warn] %s: %v\n", base, err)
				}
				return
			}
			fileRows++
		})
		if procErr != nil {
			// A file that cannot be read is logged and counted, never fatal —
			// one corrupt file must not abandon the batch.
			fmt.Fprintf(os.Stderr, "  [error] %s: %v\n", base, procErr)
			fileErrors++
		}

		st.Skipped += fileSkipped
		st.Errors += fileErrors
		fmt.Printf("  %s: %d rows\n", base, fileRows)
	}

	if err := ins.Close(); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// VACUUM only after COMMIT — SQLite refuses it inside a transaction.
	return writer.Finish(db, outputPath, st)
}

// cellAt returns the trimmed cell at idx, or "" when idx is out of range.
func cellAt(row []reader.Cell, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx].Str)
}
