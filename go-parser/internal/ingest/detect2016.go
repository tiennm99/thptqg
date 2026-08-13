package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tiennm99/thptqg/go-parser/internal/config"
	"github.com/tiennm99/thptqg/go-parser/internal/reader"
	"github.com/tiennm99/thptqg/go-parser/internal/transform"
	"github.com/tiennm99/thptqg/go-parser/internal/writer"
)

// The 2016 dataset's 119 files were produced by inconsistent tooling and use
// three different column layouts, chosen per sheet at runtime. This is a port of
// parser/src/format_detect_2016.rs — institutional knowledge encoded as
// literals, with no abstraction to derive it from, so everything here is copied
// verbatim rather than rationalised.

// KnownHeaders are the upper-cased first-cell values that identify a header row
// (format_detect_2016.rs:36-54, mirroring KNOWN_HEADERS in build-database.js).
//
// NOTE "SINH " carries a TRAILING SPACE, exactly as in the Rust source. Trimming
// it would change which rows are recognised as headers.
var KnownHeaders = []string{
	"SOBAODANH",
	"SBD",
	"HO_TEN",
	"HOTEN",
	"HỌ TÊN",
	"NGAY_SINH",
	"TEN_CUMTHI",
	"GIOI_TINH",
	"DIEM_THI",
	"STT",
	"TOAN",
	"VAN",
	"LY",
	"HOA",
	"SINH ",
	"SU",
	"DIA",
}

func isKnownHeader(s string) bool {
	for _, h := range KnownHeaders {
		if h == s {
			return true
		}
	}
	return false
}

// IsHeaderRow2016 reports whether row[0] is a known header token
// (format_detect_2016.rs:58-64).
//
// The guard here is len < 2, not the len < 3 used by the 2017 header check.
func IsHeaderRow2016(row []reader.Cell) bool {
	if len(row) < 2 {
		return false
	}
	return isKnownHeader(strings.ToUpper(strings.TrimSpace(row[0].Str)))
}

// FormatKind is one of the three 2016 layouts.
type FormatKind int

const (
	// FormatSeparateScores has one column per subject rather than a free-text
	// DIEM_THI cell — the dhhanghai-style files.
	FormatSeparateScores FormatKind = iota
	// FormatMapped resolves column indices from the header by name.
	FormatMapped
	// FormatDefault is the positional 6-column layout used when no header is
	// recognised. It is FormatMapped with fixed indices, not a separate path.
	FormatDefault
)

// Format is a detected layout plus, for the mapped case, its column indices.
type Format struct {
	Kind      FormatKind
	Sbd       int
	HoTen     int
	NgaySinh  *int
	TenCumThi *int
	GioiTinh  *int
	DiemThi   int
}

// defaultFormat is FormatMapped with the fixed positional indices
// (format_detect_2016.rs:295-306).
func defaultFormat() Format {
	two, three, four := 2, 3, 4
	return Format{
		Kind: FormatDefault, Sbd: 0, HoTen: 1,
		NgaySinh: &two, TenCumThi: &three, GioiTinh: &four, DiemThi: 5,
	}
}

// DetectFormat inspects a header row and decides which layout applies
// (format_detect_2016.rs:95-145).
func DetectFormat(headerRow []reader.Cell) Format {
	cols := make([]string, len(headerRow))
	for i, c := range headerRow {
		cols[i] = strings.ToUpper(strings.TrimSpace(c.Str))
	}

	// Format 1: SBD in col 0 AND TOAN in col 2.
	if len(cols) > 2 && cols[0] == "SBD" && cols[2] == "TOAN" {
		return Format{Kind: FormatSeparateScores}
	}

	// Format 2: resolve indices by header name, order-independent.
	var sbdIdx, hoTenIdx, ngaySinhIdx, tenCumThiIdx, gioiTinhIdx, diemThiIdx *int
	for i := range cols {
		idx := i
		switch cols[i] {
		case "SOBAODANH", "SBD":
			sbdIdx = &idx
		case "HO_TEN", "HOTEN", "HỌ TÊN":
			hoTenIdx = &idx
		case "NGAY_SINH":
			ngaySinhIdx = &idx
		case "TEN_CUMTHI":
			tenCumThiIdx = &idx
		case "GIOI_TINH":
			gioiTinhIdx = &idx
		case "DIEM_THI":
			diemThiIdx = &idx
		}
	}

	if sbdIdx != nil && diemThiIdx != nil {
		hoTen := 1 // fallback: col 1, present in all known files
		if hoTenIdx != nil {
			hoTen = *hoTenIdx
		}
		return Format{
			Kind: FormatMapped, Sbd: *sbdIdx, HoTen: hoTen,
			NgaySinh: ngaySinhIdx, TenCumThi: tenCumThiIdx, GioiTinh: gioiTinhIdx,
			DiemThi: *diemThiIdx,
		}
	}

	// Unrecognised header, or none at all.
	return defaultFormat()
}

// parseFloatCell parses a per-subject score cell.
//
// A parsed 0.0 becomes "no score" — this replicates JavaScript's
// `parseFloat(row[N]) || null`, where 0 is falsy (format_detect_2016.rs:165).
// It means a genuine zero is indistinguishable from a blank. Not obviously
// correct, but it is the shipped behaviour and the published data depends on it.
func parseFloatCell(row []reader.Cell, idx int) (float64, bool) {
	s := cellAt(row, idx)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v == 0.0 {
		return 0, false
	}
	return v, true
}

// processSeparateScoresRow handles the fixed 12-column layout
// (format_detect_2016.rs:176-216):
//
//	0=SBD 1=HOTEN 2=TOAN 3=VAN 4=LY 5=HOA 6=SINH 7=SU 8=DIA
//	9=NGOAINGUTN 10=NGOAINGUTL 11=NGOAINGU(total -> tieng_anh)
//
// tieng_phap / tieng_duc / tieng_nhat / tieng_trung are structurally unreachable
// in this format, and ngay_sinh / ten_cum_thi / gioi_tinh are always nil.
// Scores are read as floats directly; this layout has no free-text score cell,
// so the subject regexes never run.
func processSeparateScoresRow(row []reader.Cell) *transform.ParsedRow {
	sbd := cellAt(row, 0)
	hoTen := cellAt(row, 1)
	if sbd == "" || hoTen == "" {
		return nil
	}

	scores := make(map[string]float64)
	for field, idx := range map[string]int{
		"toan": 2, "ngu_van": 3, "vat_ly": 4, "hoa_hoc": 5,
		"sinh_hoc": 6, "lich_su": 7, "dia_ly": 8,
		"tieng_anh": 11, // NGOAINGU total
	} {
		if v, ok := parseFloatCell(row, idx); ok {
			scores[field] = v
		}
	}

	return &transform.ParsedRow{
		SoBaoDanh:  sbd,
		HoTen:      hoTen,
		HoTenAscii: transform.ToAscii(hoTen),
		Scores:     scores,
	}
}

// processMappedRow handles header-derived column indices
// (format_detect_2016.rs:226-289).
func processMappedRow(row []reader.Cell, f Format) *transform.ParsedRow {
	sbd := cellAt(row, f.Sbd)
	hoTen := cellAt(row, f.HoTen)
	if sbd == "" || hoTen == "" {
		return nil
	}

	// Leaked-header guard: a row whose SBD or name cell is itself a header token
	// is a repeated header, not data (format_detect_2016.rs:244-250).
	if isKnownHeader(strings.ToUpper(sbd)) || isKnownHeader(strings.ToUpper(hoTen)) {
		return nil
	}

	optional := func(idx *int) *string {
		if idx == nil {
			return nil
		}
		if s := cellAt(row, *idx); s != "" {
			return &s
		}
		return nil
	}

	// Gender is a two-value allowlist, not a general enum: anything other than
	// exactly "Nam" or "Nữ" becomes nil (format_detect_2016.rs:263-271).
	var gioiTinh *string
	if f.GioiTinh != nil {
		if s := cellAt(row, *f.GioiTinh); s == "Nam" || s == "Nữ" {
			gioiTinh = &s
		}
	}

	// Read untrimmed, matching format_detect_2016.rs:273-276.
	diemThi := ""
	if f.DiemThi >= 0 && f.DiemThi < len(row) {
		diemThi = row[f.DiemThi].Str
	}

	return &transform.ParsedRow{
		SoBaoDanh:  sbd,
		HoTen:      hoTen,
		HoTenAscii: transform.ToAscii(hoTen),
		NgaySinh:   optional(f.NgaySinh),
		TenCumThi:  optional(f.TenCumThi),
		GioiTinh:   gioiTinh,
		Scores:     transform.ParseScores(diemThi),
	}
}

// ProcessRow2016 dispatches a data row through the detected layout. A nil return
// means the row is empty or invalid and should be skipped.
func ProcessRow2016(row []reader.Cell, f Format) *transform.ParsedRow {
	if f.Kind == FormatSeparateScores {
		return processSeparateScoresRow(row)
	}
	// FormatMapped and FormatDefault share one implementation; Default is just a
	// fixed index tuple.
	return processMappedRow(row, f)
}

// Detect2016 ingests the 2016 dataset — the port of run_build_2016 and
// process_file_2016 (main.rs:211-377).
func Detect2016(cfg *config.DatasetConfig, inputDir, outputPath string) error {
	files, err := InputFiles(inputDir)
	if err != nil {
		return err
	}
	label := DatasetLabel(inputDir)
	fmt.Printf("[build:2016] %s/ → %s  (%d files)\n", label, outputPath, len(files))

	db, err := writer.OpenDB(outputPath)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	ins, err := writer.Prepare(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	var st writer.Stats // Skipped stays 0 on this path, matching main.rs:251
	for _, file := range files {
		base := filepath.Base(file)
		fileRows, err := processFile2016(file, cfg, ins, base, &st)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [error] %s: %v\n", base, err)
			st.Errors++
			continue
		}
		fmt.Printf("  %s: %d rows\n", base, fileRows)
	}

	if err := ins.Close(); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return writer.Finish(db, outputPath, st, label, false)
}

// processFile2016 detects the layout per SHEET and processes that sheet's rows.
//
// Detection is per sheet, not per file (main.rs:344-349): within one workbook,
// sheet 2 may legitimately detect differently from sheet 1, which matters for
// the province files that overflow past Excel's row cap.
func processFile2016(path string, cfg *config.DatasetConfig, ins *writer.Inserter, base string, st *writer.Stats) (uint64, error) {
	wb, err := reader.Open(path)
	if err != nil {
		return 0, err
	}
	defer wb.Close()

	sheets := wb.Sheets()
	if len(sheets) == 0 {
		return 0, nil
	}
	if cfg.Reader.SheetMode == config.SheetModeFirst {
		sheets = sheets[:1]
	}

	var fileRows uint64
	for _, sh := range sheets {
		var rows [][]reader.Cell
		if err := wb.EachRow(sh.Index, func(_ reader.Sheet, _ int, row []reader.Cell) error {
			rows = append(rows, row)
			return nil
		}); err != nil {
			return fileRows, err
		}
		if len(rows) == 0 {
			continue
		}

		f := defaultFormat()
		startIdx := 0
		if IsHeaderRow2016(rows[0]) {
			f = DetectFormat(rows[0])
			startIdx = 1
		}

		for _, row := range rows[startIdx:] {
			// Rows shorter than 2 cells are dropped BEFORE the counter
			// (main.rs:351-353), so they never appear in the source total.
			if len(row) < 2 {
				continue
			}
			st.SourceRows++

			parsed := ProcessRow2016(row, f)
			if parsed == nil {
				// Empty or invalid. Note this is NOT counted as skipped — the
				// 2016 path leaves that counter at zero (main.rs:251), so the
				// stats block reports insertable == source rows and the Audit
				// line absorbs the difference.
				continue
			}
			if err := ins.Insert(parsed); err != nil {
				st.Errors++
				if st.Errors <= 5 {
					fmt.Fprintf(os.Stderr, "  [warn] %s: %v\n", base, err)
				}
				continue
			}
			fileRows++
		}
	}
	return fileRows, nil
}
