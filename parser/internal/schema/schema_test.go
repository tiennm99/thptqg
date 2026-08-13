package schema

import (
	"strings"
	"testing"
)

// Ports the four tests in parser/src/schema.rs:149-213. Their purpose is to stop
// the DDL, the INSERT column list and the field-order constants from drifting
// apart — a drift that silently lands values in the wrong columns.

// TestInsertMatchesFieldOrder ports insert_matches_field_order (schema.rs:156).
func TestInsertMatchesFieldOrder(t *testing.T) {
	if ParamCount != 22 {
		t.Errorf("ParamCount = %d, want 22", ParamCount)
	}
	if got := strings.Count(InsertSQL, "?"); got != ParamCount {
		t.Errorf("INSERT placeholders = %d, want %d", got, ParamCount)
	}

	open := strings.Index(InsertSQL, "(")
	closeIdx := strings.Index(InsertSQL, ")")
	if open < 0 || closeIdx < 0 {
		t.Fatal("INSERT must contain a column list")
	}
	var listed []string
	for _, c := range strings.Split(InsertSQL[open+1:closeIdx], ",") {
		if c = strings.TrimSpace(c); c != "" {
			listed = append(listed, c)
		}
	}

	want := append(append([]string{}, IdentityFields...), ScoreFields...)
	if len(listed) != len(want) {
		t.Fatalf("INSERT lists %d columns, want %d", len(listed), len(want))
	}
	for i := range want {
		if listed[i] != want[i] {
			t.Errorf("column %d: INSERT has %q, field order has %q", i, listed[i], want[i])
		}
	}
}

// TestScorePatternsCoverScoreFields ports score_patterns_cover_score_fields
// (schema.rs:183).
func TestScorePatternsCoverScoreFields(t *testing.T) {
	if len(ScorePatterns) != len(ScoreFields) {
		t.Fatalf("%d patterns for %d score columns", len(ScorePatterns), len(ScoreFields))
	}
	inFields := make(map[string]bool, len(ScoreFields))
	for _, f := range ScoreFields {
		inFields[f] = true
	}
	for field := range ScorePatterns {
		if !inFields[field] {
			t.Errorf("pattern %q has no column", field)
		}
	}
	for _, field := range ScoreFields {
		if _, ok := ScorePatterns[field]; !ok {
			t.Errorf("column %q has no pattern", field)
		}
	}
}

// TestDDLColumnsMatchInsert ports ddl_columns_match_insert (schema.rs:201).
func TestDDLColumnsMatchInsert(t *testing.T) {
	for _, field := range append(append([]string{}, IdentityFields...), ScoreFields...) {
		if !strings.Contains(DDL, field) {
			t.Errorf("DDL missing column %q", field)
		}
	}
}

// TestScorePatternsCompile ports score_patterns_compile (schema.rs:208).
// Compilation happens in the package initialiser, so reaching this point already
// proves it; the explicit checks guard against an empty or partial table.
func TestScorePatternsCompile(t *testing.T) {
	for field, re := range ScorePatterns {
		if re == nil {
			t.Errorf("pattern %q is nil", field)
		}
	}
}

// TestDDLMatchesRust asserts the DDL is byte-identical to parser/src/schema.rs.
// Anything less and the two parsers can produce structurally different databases
// while every row-level check still passes.
func TestDDLMatchesRust(t *testing.T) {
	const want = `
CREATE TABLE student (
  so_bao_danh   TEXT PRIMARY KEY,
  ho_ten        TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,
  ngay_sinh     TEXT,
  ten_cum_thi   TEXT,
  gioi_tinh     TEXT,
  toan          REAL,
  ngu_van       REAL,
  vat_ly        REAL,
  hoa_hoc       REAL,
  sinh_hoc      REAL,
  khtn          REAL,
  lich_su       REAL,
  dia_ly        REAL,
  gdcd          REAL,
  khxh          REAL,
  tieng_anh     REAL,
  tieng_phap    REAL,
  tieng_nga     REAL,
  tieng_duc     REAL,
  tieng_nhat    REAL,
  tieng_trung   REAL
);
CREATE INDEX idx_ho_ten       ON student(ho_ten);
CREATE INDEX idx_ho_ten_ascii ON student(ho_ten_ascii);
CREATE INDEX idx_ten_cum_thi  ON student(ten_cum_thi) WHERE ten_cum_thi IS NOT NULL;
`
	if DDL != want {
		t.Errorf("DDL diverges from parser/src/schema.rs:26-54\n--- got ---\n%s\n--- want ---\n%s", DDL, want)
	}
}

// TestScorePatternsMatchScores exercises each pattern against the shape the
// DIEM_THI cell actually carries, including the wide runs of spaces seen in the
// real corpus.
func TestScorePatternsMatchScores(t *testing.T) {
	const cell = "Toán:   8.50   Ngữ văn:   7.00   Tiếng Đức:   9   KHXH: 5.58   "
	cases := map[string]string{
		"toan":       "8.50",
		"ngu_van":    "7.00",
		"tieng_duc":  "9",
		"khxh":       "5.58",
		"tieng_nhat": "", // absent from the cell -> no match
	}
	for field, want := range cases {
		re, ok := ScorePatterns[field]
		if !ok {
			t.Fatalf("no pattern for %q", field)
		}
		m := re.FindStringSubmatch(cell)
		got := ""
		if m != nil {
			got = m[1]
		}
		if got != want {
			t.Errorf("%s: matched %q, want %q", field, got, want)
		}
	}
}
