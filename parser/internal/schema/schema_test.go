package schema

import (
	"strings"
	"testing"
)

// These tests stop the DDL, the INSERT column list and the field-order constants
// from drifting apart — a drift that silently lands values in the wrong columns.

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

func TestDDLColumnsMatchInsert(t *testing.T) {
	for _, field := range append(append([]string{}, IdentityFields...), ScoreFields...) {
		if !strings.Contains(DDL, field) {
			t.Errorf("DDL missing column %q", field)
		}
	}
}

// TestScorePatternsCompile: compilation happens in the package initialiser, so
// reaching this point already proves it; the explicit checks guard against an
// empty or partial table.
func TestScorePatternsCompile(t *testing.T) {
	for field, re := range ScorePatterns {
		if re == nil {
			t.Errorf("pattern %q is nil", field)
		}
	}
}

// TestDDLIsFrozen compares DDL against an independent copy of the exact text,
// down to the byte. It catches column, type and index changes that every
// row-level check would still pass — a database can be structurally different
// and look fine one row at a time. Update the copy below only when the schema
// change is intended.
func TestDDLIsFrozen(t *testing.T) {
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
`
	if DDL != want {
		t.Errorf("DDL changed\n--- got ---\n%s\n--- want ---\n%s", DDL, want)
	}
}

// TestNoSecondaryIndexes: the browser downloads this file whole and queries it
// in memory, where a scan of 877,460 rows takes a few hundred milliseconds. An
// index would save that and cost every visitor tens of megabytes of download.
func TestNoSecondaryIndexes(t *testing.T) {
	if strings.Contains(DDL, "CREATE INDEX") {
		t.Error("DDL creates an index; every one of them is paid for on the network")
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
