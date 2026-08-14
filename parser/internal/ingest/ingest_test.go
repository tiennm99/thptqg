package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tiennm99/thptqg/parser/internal/reader"
)

func cells(vals ...string) []reader.Cell {
	out := make([]reader.Cell, len(vals))
	for i, v := range vals {
		out[i] = reader.Cell{Str: v, IsEmpty: v == ""}
	}
	return out
}

var stdTokens = []string{"HO_TEN", "HỌ TÊN", "STT"}

func TestHeaderDetectsHoTen(t *testing.T) {
	if !IsHeaderRow(cells("HO_TEN", "NGAY_SINH", "SBD"), stdTokens) {
		t.Error("HO_TEN header not detected")
	}
}

func TestHeaderDetectsStt(t *testing.T) {
	if !IsHeaderRow(cells("STT", "B", "C"), stdTokens) {
		t.Error("STT header not detected")
	}
}

func TestHeaderDetectsHoTenUnicode(t *testing.T) {
	if !IsHeaderRow(cells("HỌ TÊN", "B", "C"), stdTokens) {
		t.Error("HỌ TÊN header not detected")
	}
}

func TestHeaderRejectsDataRow(t *testing.T) {
	if IsHeaderRow(cells("Nguyen Van A", "01/01/2000", "12345678"), stdTokens) {
		t.Error("data row wrongly detected as header")
	}
}

// TestHeaderRejectsShortRow covers the <3-cell guard.
func TestHeaderRejectsShortRow(t *testing.T) {
	if IsHeaderRow(cells("HO_TEN", ""), stdTokens) {
		t.Error("a 2-cell row must never be a header, even with a matching token")
	}
}

func TestHeaderCaseInsensitive(t *testing.T) {
	if !IsHeaderRow(cells("ho_ten", "B", "C"), stdTokens) {
		t.Error("lowercase header not detected")
	}
}

// TestBlankRowDetection: the third cell is an empty *string* cell rather than an
// absent one, and must still count as blank.
func TestBlankRowDetection(t *testing.T) {
	blank := []reader.Cell{{IsEmpty: true}, {IsEmpty: true}, {Str: ""}}
	if !IsAllBlank(blank) {
		t.Error("row of empty cells should be blank")
	}
	if IsAllBlank(cells("Nguyen", "", "")) {
		t.Error("row with content should not be blank")
	}
}

// TestIsAllBlankIgnoresIsEmptyFlag pins the rule that blankness is decided by
// the rendered string, never by Cell.IsEmpty, which is diagnostic only.
func TestIsAllBlankIgnoresIsEmptyFlag(t *testing.T) {
	// Content present but IsEmpty wrongly set: still not blank.
	if IsAllBlank([]reader.Cell{{Str: "x", IsEmpty: true}}) {
		t.Error("a cell with content must not be blank regardless of IsEmpty")
	}
	// Whitespace only: blank.
	if !IsAllBlank([]reader.Cell{{Str: "   "}, {Str: "\t"}}) {
		t.Error("whitespace-only cells should be blank")
	}
}

// TestDatasetLabel covers the basename derivation that drives the stats wording.
func TestDatasetLabel(t *testing.T) {
	for in, want := range map[string]string{
		"data/2017":       "2017",
		"data/2017-old2":  "2017-old2",
		"data/2017-old2/": "2017-old2",
		"/abs/path/2016":  "2016",
	} {
		if got := DatasetLabel(in); got != want {
			t.Errorf("DatasetLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestInputFilesSortedAndFiltered: the sort decides which duplicate SBD survives
// INSERT OR REPLACE, so it is behaviour, not presentation.
func TestInputFilesSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.xlsx", "a.xls", "c.XLSX", "notes.txt", "d.csv"} {
		if err := writeEmpty(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := InputFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.xls", "b.xlsx", "c.XLSX"}
	if len(got) != len(want) {
		t.Fatalf("got %d files %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if filepath.Base(got[i]) != want[i] {
			t.Errorf("file %d = %q, want %q", i, filepath.Base(got[i]), want[i])
		}
	}
}

func writeEmpty(path string) error { return os.WriteFile(path, nil, 0o644) }
