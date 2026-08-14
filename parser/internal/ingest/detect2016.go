package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tiennm99/thptqg/parser/internal/config"
	"github.com/tiennm99/thptqg/parser/internal/reader"
	"github.com/tiennm99/thptqg/parser/internal/transform"
	"github.com/tiennm99/thptqg/parser/internal/writer"
)

// The 2016 dataset's 119 files were produced by inconsistent tooling and use
// four different column layouts, chosen per sheet at runtime. The literals
// below are institutional knowledge about those files, with no rule to derive
// them from — extend them, but do not tidy them into something more regular.

// KnownHeaders are the upper-cased first-cell values that identify a header row.
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
	"SINH",
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

// IsHeaderRow2016 reports whether row[0] is a known header token.
//
// The guard here is len < 2, not the len < 3 used by the 2017 header check.
func IsHeaderRow2016(row []reader.Cell) bool {
	if len(row) < 2 {
		return false
	}
	return isKnownHeader(strings.ToUpper(strings.TrimSpace(row[0].Str)))
}

// FormatKind is one of the 2016 layouts.
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
	// FormatSubjectColumns resolves BOTH identity and one score column per
	// subject from the header by name — the university-cluster files, which
	// publish scores in columns instead of a DIEM_THI sentence.
	FormatSubjectColumns
)

// Format is a detected layout plus its resolved column indices.
type Format struct {
	Kind      FormatKind
	Sbd       int
	HoTen     int
	NgaySinh  *int
	TenCumThi *int
	GioiTinh  *int
	DiemThi   int

	// Subjects maps a schema score field to its column, for
	// FormatSubjectColumns only.
	Subjects map[string]int
	// LangScore is the foreign-language score column; the subject it counts
	// towards is named by the code in LangCode.
	LangScore *int
	LangCode  *int
}

// defaultFormat is FormatMapped with the fixed positional indices.
func defaultFormat() Format {
	two, three, four := 2, 3, 4
	return Format{
		Kind: FormatDefault, Sbd: 0, HoTen: 1,
		NgaySinh: &two, TenCumThi: &three, GioiTinh: &four, DiemThi: 5,
	}
}

// identity holds the identity columns resolved from a header row. A nil field
// means the header does not name that column.
type identity struct {
	Sbd, HoTen, NgaySinh, TenCumThi, GioiTinh, DiemThi *int
}

// resolveIdentity maps header names to identity columns, order-independent.
// Each column has several spellings across the corpus, including the accented
// and unseparated ones the university files use.
func resolveIdentity(cols []string) identity {
	var id identity
	for i := range cols {
		idx := i
		switch cols[i] {
		case "SOBAODANH", "SBD":
			id.Sbd = &idx
		case "HO_TEN", "HOTEN", "HỌ TÊN":
			id.HoTen = &idx
		case "NGAY_SINH", "NGAYSINH", "NGÀY SINH":
			id.NgaySinh = &idx
		case "TEN_CUMTHI":
			id.TenCumThi = &idx
		case "GIOI_TINH", "GIỚI TÍNH", "PHAI":
			id.GioiTinh = &idx
		case "DIEM_THI":
			id.DiemThi = &idx
		}
	}
	return id
}

// subjectColumnFamily is one university-cluster spelling of the per-subject
// score columns. The two families are told apart by which names resolve.
type subjectColumnFamily struct {
	scores    map[string]string // header name -> schema score field
	langScore string            // header of the foreign-language score column
	langCode  string            // header of the N1..N6 language code column
}

// subjectColumnFamilies are tried in order. The Cần Thơ family goes first
// because its CDIEM<n> names are unambiguous, while the two-letter family
// shares "HO" with that layout's surname column.
var subjectColumnFamilies = []subjectColumnFamily{
	{scores: canThoScoreHeaders, langScore: "CDIEM2", langCode: "NGOAINGU"},
	{scores: subjectCodeHeaders, langScore: "NN", langCode: "MÔN NN"},
}

// canThoScoreHeaders maps the Cần Thơ cluster's published-score columns to
// schema fields. The columns are numbered in the order the 2016 exam was sat —
// each morning an essay paper (quarter-point scores), each afternoon a
// multiple-choice one — which is what identifies them: CDIEM1/3/5/7 quantise to
// 0.25 and CDIEM2/4/6/8 do not, and the per-column means match the published
// national averages for those subjects.
var canThoScoreHeaders = map[string]string{
	"CDIEM1": "toan",
	"CDIEM3": "ngu_van",
	"CDIEM4": "vat_ly",
	"CDIEM5": "dia_ly",
	"CDIEM6": "hoa_hoc",
	"CDIEM7": "lich_su",
	"CDIEM8": "sinh_hoc",
}

// subjectCodeHeaders maps the two-letter subject columns to schema fields.
var subjectCodeHeaders = map[string]string{
	"TO": "toan",
	"VA": "ngu_van",
	"LI": "vat_ly",
	"HO": "hoa_hoc",
	"SI": "sinh_hoc",
	"SU": "lich_su",
	"DI": "dia_ly",
}

// languageFields maps the ministry's foreign-language codes to schema fields.
var languageFields = map[string]string{
	"N1": "tieng_anh",
	"N2": "tieng_nga",
	"N3": "tieng_phap",
	"N4": "tieng_trung",
	"N5": "tieng_duc",
	"N6": "tieng_nhat",
}

// DetectFormat inspects a header row and decides which layout applies.
func DetectFormat(headerRow []reader.Cell) Format {
	cols := make([]string, len(headerRow))
	for i, c := range headerRow {
		cols[i] = strings.ToUpper(strings.TrimSpace(c.Str))
	}

	// Format 1: SBD in col 0 AND TOAN in col 2.
	if len(cols) > 2 && cols[0] == "SBD" && cols[2] == "TOAN" {
		return Format{Kind: FormatSeparateScores}
	}

	id := resolveIdentity(cols)

	// Format 2: a free-text DIEM_THI cell alongside an SBD.
	if id.Sbd != nil && id.DiemThi != nil {
		hoTen := 1 // fallback: col 1, present in all known files
		if id.HoTen != nil {
			hoTen = *id.HoTen
		}
		return Format{
			Kind: FormatMapped, Sbd: *id.Sbd, HoTen: hoTen,
			NgaySinh: id.NgaySinh, TenCumThi: id.TenCumThi, GioiTinh: id.GioiTinh,
			DiemThi: *id.DiemThi,
		}
	}

	// Format 3: one column per subject.
	if f, ok := subjectColumnsFormat(cols, id); ok {
		return f
	}

	// Unrecognised header, or none at all.
	return defaultFormat()
}

// subjectColumnsFormat resolves a per-subject-column layout, if the header
// names one.
//
// Three resolved subjects are required: a stray one- or two-letter header in an
// unrelated file must not turn that file into this layout.
func subjectColumnsFormat(cols []string, id identity) (Format, bool) {
	if id.Sbd == nil {
		return Format{}, false
	}
	for _, fam := range subjectColumnFamilies {
		subjects := make(map[string]int)
		for i, name := range cols {
			if field, ok := fam.scores[name]; ok {
				subjects[field] = i
			}
		}
		if len(subjects) < 3 {
			continue
		}
		hoTen := 1
		if id.HoTen != nil {
			hoTen = *id.HoTen
		}
		return Format{
			Kind: FormatSubjectColumns, Sbd: *id.Sbd, HoTen: hoTen,
			NgaySinh: id.NgaySinh, TenCumThi: id.TenCumThi, GioiTinh: id.GioiTinh,
			DiemThi:   -1,
			Subjects:  subjects,
			LangScore: indexOf(cols, fam.langScore),
			LangCode:  indexOf(cols, fam.langCode),
		}, true
	}
	return Format{}, false
}

// indexOf returns the column holding name, or nil when the header lacks it.
func indexOf(cols []string, name string) *int {
	for i := range cols {
		if cols[i] == name {
			idx := i
			return &idx
		}
	}
	return nil
}

// parseScoreCell parses a per-subject score cell. A parsed 0 is a real score:
// the candidate sat the paper and scored nothing.
func parseScoreCell(row []reader.Cell, idx int) (float64, bool) {
	s := cellAt(row, idx)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// processSeparateScoresRow handles the fixed 12-column layout:
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
		if v, ok := parseScoreCell(row, idx); ok {
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

// processMappedRow handles header-derived column indices.
func processMappedRow(row []reader.Cell, f Format) *transform.ParsedRow {
	sbd := cellAt(row, f.Sbd)
	hoTen := cellAt(row, f.HoTen)
	if sbd == "" || hoTen == "" {
		return nil
	}

	// Leaked-header guard: a row whose SBD or name cell is itself a header token
	// is a repeated header, not data.
	if isKnownHeader(strings.ToUpper(sbd)) || isKnownHeader(strings.ToUpper(hoTen)) {
		return nil
	}

	return &transform.ParsedRow{
		SoBaoDanh:  sbd,
		HoTen:      hoTen,
		HoTenAscii: transform.ToAscii(hoTen),
		NgaySinh:   birthDate(row, f.NgaySinh),
		TenCumThi:  optionalCell(row, f.TenCumThi),
		GioiTinh:   gender(row, f.GioiTinh),
		Scores:     transform.ParseScores(cellAt(row, f.DiemThi)),
	}
}

// processSubjectColumnsRow reads a row whose scores sit in one column per
// subject, with the foreign language filed under the subject its code names.
func processSubjectColumnsRow(row []reader.Cell, f Format) *transform.ParsedRow {
	sbd := cellAt(row, f.Sbd)
	hoTen := cellAt(row, f.HoTen)
	if sbd == "" || hoTen == "" {
		return nil
	}
	if isKnownHeader(strings.ToUpper(sbd)) || isKnownHeader(strings.ToUpper(hoTen)) {
		return nil
	}

	scores := make(map[string]float64)
	for field, idx := range f.Subjects {
		if v, ok := parseScoreCell(row, idx); ok {
			scores[field] = v
		}
	}
	if f.LangScore != nil {
		if v, ok := parseScoreCell(row, *f.LangScore); ok {
			code := "N1" // an unlabelled language paper is English in this corpus
			if f.LangCode != nil {
				if c := strings.ToUpper(cellAt(row, *f.LangCode)); c != "" {
					code = c
				}
			}
			// An unknown code is left out rather than guessed at.
			if field, ok := languageFields[code]; ok {
				scores[field] = v
			}
		}
	}

	return &transform.ParsedRow{
		SoBaoDanh:  sbd,
		HoTen:      hoTen,
		HoTenAscii: transform.ToAscii(hoTen),
		NgaySinh:   birthDate(row, f.NgaySinh),
		TenCumThi:  optionalCell(row, f.TenCumThi),
		GioiTinh:   gender(row, f.GioiTinh),
		Scores:     scores,
	}
}

// optionalCell returns the trimmed cell at idx, or nil when the column is
// absent or its cell empty.
func optionalCell(row []reader.Cell, idx *int) *string {
	if idx == nil {
		return nil
	}
	if s := cellAt(row, *idx); s != "" {
		return &s
	}
	return nil
}

// gender normalises the gender cell to "Nam"/"Nữ", or nil when it says neither.
//
// The Cần Thơ files encode it numerically instead: of the rows marked 1, 53%
// carry the female marker "Thị" in the name, against 1% of those marked 0.
func gender(row []reader.Cell, idx *int) *string {
	s := optionalCell(row, idx)
	if s == nil {
		return nil
	}
	switch *s {
	case "Nam", "Nữ":
		return s
	case "0":
		nam := "Nam"
		return &nam
	case "1":
		nu := "Nữ"
		return &nu
	}
	return nil
}

// birthDate returns the birth-date cell as dd/mm/yyyy.
//
// Most files already write it that way; the Cần Thơ ones use a compact ddmmyy,
// expanded here so the column holds one format. The century is always 19xx: a
// 2016 candidate born after 1999 would have sat the exam under age.
func birthDate(row []reader.Cell, idx *int) *string {
	s := optionalCell(row, idx)
	if s == nil || len(*s) != 6 || !allDigits(*s) {
		return s
	}
	out := (*s)[0:2] + "/" + (*s)[2:4] + "/19" + (*s)[4:6]
	return &out
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// headerScanRows is how far into a sheet the header may sit. One file opens
// with a three-row ministry title block above its header; a sheet with no
// header at all has data from row 0 and simply finds nothing in that window.
const headerScanRows = 5

// sheetFormat picks the layout for one sheet and returns the row its data
// starts on. Rows above the header are a title block and are not data.
func sheetFormat(rows [][]reader.Cell) (Format, int) {
	limit := headerScanRows
	if len(rows) < limit {
		limit = len(rows)
	}
	for i := 0; i < limit; i++ {
		if IsHeaderRow2016(rows[i]) {
			return DetectFormat(rows[i]), i + 1
		}
	}
	return defaultFormat(), 0
}

// ProcessRow2016 dispatches a data row through the detected layout. A nil return
// means the row is empty or invalid and should be skipped.
func ProcessRow2016(row []reader.Cell, f Format) *transform.ParsedRow {
	switch f.Kind {
	case FormatSeparateScores:
		return processSeparateScoresRow(row)
	case FormatSubjectColumns:
		return processSubjectColumnsRow(row, f)
	}
	// FormatMapped and FormatDefault share one implementation; Default is just a
	// fixed index tuple.
	return processMappedRow(row, f)
}

// Detect2016 ingests the 2016 dataset.
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

	var st writer.Stats // Skipped stays 0 on this path
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
	return writer.Finish(db, outputPath, st)
}

// processFile2016 detects the layout per SHEET and processes that sheet's rows.
//
// Detection is per sheet, not per file: within one workbook, sheet 2 may
// legitimately detect differently from sheet 1, which matters for the province
// files that overflow past Excel's row cap.
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

		f, startIdx := sheetFormat(rows)

		for _, row := range rows[startIdx:] {
			// Rows shorter than 2 cells are dropped BEFORE the counter, so they
			// never appear in the source total.
			if len(row) < 2 {
				continue
			}
			st.SourceRows++

			parsed := ProcessRow2016(row, f)
			if parsed == nil {
				// Empty or invalid. Note this is NOT counted as skipped — the
				// 2016 path leaves that counter at zero, so the stats block
				// reports insertable == source rows and the Audit line absorbs
				// the difference.
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
