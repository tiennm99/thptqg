// Package audit compares distinct SBDs in the source spreadsheets against the
// row count in a built database.
//
// Two divergences from the build path are deliberate:
//
//   - only .xlsx files are considered
//   - only sheet 0 is read, regardless of sheet_mode
//
// They are intentional, not oversights. Do not "fix" them.
package audit

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tiennm99/thptqg/parser/internal/config"
	"github.com/tiennm99/thptqg/parser/internal/ingest"
	"github.com/tiennm99/thptqg/parser/internal/reader"
	"github.com/tiennm99/thptqg/parser/internal/sqlitedb"
)

// Result carries the audit counters.
type Result struct {
	TotalDataRows uint64
	BothEmpty     uint64
	EmptyName     uint64
	EmptySbd      uint64
	DistinctSbds  int
	DBCount       int64
	Matched       bool
}

// Run collects distinct SBDs from the .xlsx files in inputDir and compares the
// total against the student row count in dbPath.
func Run(inputDir, dbPath string, cfg *config.DatasetConfig) (*Result, error) {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, fmt.Errorf("read input dir %s: %w", inputDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".xlsx") {
			files = append(files, filepath.Join(inputDir, e.Name()))
		}
	}
	sort.Strings(files)

	res := &Result{}
	seen := make(map[string]struct{})

	// Format-detection configs carry no Columns block, so the audit falls back
	// to fixed positional columns for them.
	hoTenCol, sbdCol := 1, 0
	if cfg.Columns != nil {
		hoTenCol, sbdCol = cfg.Columns.HoTen, cfg.Columns.SoBaoDanh
	}

	for _, file := range files {
		wb, err := reader.Open(file)
		if err != nil {
			return nil, err
		}
		sheets := wb.Sheets()
		if len(sheets) == 0 {
			wb.Close()
			continue
		}

		firstRow := true
		err = wb.EachRow(sheets[0].Index, func(_ reader.Sheet, _ int, row []reader.Cell) error {
			if firstRow {
				firstRow = false
				if ingest.IsHeaderRow(row, cfg.Header.Tokens) {
					return nil
				}
			}
			res.TotalDataRows++

			hoTen := cellAt(row, hoTenCol)
			sbd := cellAt(row, sbdCol)

			if hoTen == "" && sbd == "" {
				res.BothEmpty++
				return nil
			}
			if hoTen == "" {
				res.EmptyName++
			}
			if sbd == "" {
				res.EmptySbd++
			}
			if sbd != "" {
				seen[sbd] = struct{}{}
			}
			return nil
		})
		wb.Close()
		if err != nil {
			return nil, err
		}
	}

	// Opened read-only: the audit must never mutate the database it inspects.
	db, err := sql.Open(sqlitedb.DriverName, "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open db %s: %w", dbPath, err)
	}
	defer db.Close()
	if err := db.QueryRow("SELECT COUNT(*) FROM student").Scan(&res.DBCount); err != nil {
		return nil, fmt.Errorf("count rows: %w", err)
	}

	res.DistinctSbds = len(seen)
	res.Matched = int64(res.DistinctSbds) == res.DBCount
	return res, nil
}

// PrintReport writes the source-versus-database comparison block.
func PrintReport(r *Result) {
	fmt.Println("=== Source vs DB ===")
	fmt.Printf("Source: total data rows across all files: %d\n", r.TotalDataRows)
	fmt.Printf("Source: rows with empty name AND sbd (skipped): %d\n", r.BothEmpty)
	fmt.Printf("Source: rows with missing name only: %d\n", r.EmptyName)
	fmt.Printf("Source: rows with missing sbd only: %d\n", r.EmptySbd)
	fmt.Printf("Source: distinct SBDs: %d\n", r.DistinctSbds)
	fmt.Printf("DB:     row count: %d\n", r.DBCount)
	if r.Matched {
		fmt.Println("Match:  YES — all unique SBDs accounted for")
	} else {
		fmt.Printf("Match:  NO — gap of %d\n", int64(r.DistinctSbds)-r.DBCount)
	}
}

func cellAt(row []reader.Cell, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx].Str)
}
