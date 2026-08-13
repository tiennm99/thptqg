package verify

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

// makeDB writes a miniature student table so the comparison logic can be
// exercised without building a real 877k-row database.
func makeDB(t *testing.T, dir, id string, rows [][3]any) {
	t.Helper()
	path := filepath.Join(dir, id+".db")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(driverName, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE student (
		so_bao_danh TEXT PRIMARY KEY,
		ho_ten TEXT NOT NULL,
		toan REAL
	); CREATE INDEX idx_ho_ten ON student(ho_ten);`); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if _, err := db.Exec("INSERT INTO student VALUES (?,?,?)", r[0], r[1], r[2]); err != nil {
			t.Fatal(err)
		}
	}
}

var base = [][3]any{
	{"001", "Nguyễn Văn A", 8.0},
	{"002", "Trần Thị B", 7.25},
	{"003", "Lê Văn C", nil},
}

func compare(t *testing.T, a, b [][3]any) Result {
	t.Helper()
	dirA, dirB := t.TempDir(), t.TempDir()
	makeDB(t, dirA, "x", a)
	makeDB(t, dirB, "x", b)

	res, err := Compare([]registry.Dataset{{ID: "x"}}, dirA, dirB)
	if err != nil {
		t.Fatal(err)
	}
	return res[0]
}

func TestIdenticalDatabasesMatch(t *testing.T) {
	got := compare(t, base, base)
	if !got.OK() {
		t.Fatalf("identical databases reported problems: %v", got.Problems)
	}
	if got.HashA != got.HashB || got.HashA == "" {
		t.Errorf("hashes should be equal and non-empty: %q %q", got.HashA, got.HashB)
	}
	if got.RowsA != 3 {
		t.Errorf("RowsA = %d, want 3", got.RowsA)
	}
}

// TestOneChangedScoreIsCaught is the whole point: a single altered field, with
// the row count and every non-NULL count unchanged, must still fail.
func TestOneChangedScoreIsCaught(t *testing.T) {
	changed := [][3]any{{"001", "Nguyễn Văn A", 8.25}, base[1], base[2]}
	got := compare(t, base, changed)

	if got.OK() {
		t.Fatal("a changed score was not detected")
	}
	if got.RowsA != got.RowsB {
		t.Error("row counts should still match — that is what makes this hard")
	}
	joined := strings.Join(got.Problems, "; ")
	if !strings.Contains(joined, "full-table hash differs") {
		t.Errorf("problems = %v", got.Problems)
	}
	// The diagnosis must name the row and the column.
	diff := strings.Join(got.FirstDiff, "\n")
	if !strings.Contains(diff, "so_bao_danh=001") || !strings.Contains(diff, "toan") {
		t.Errorf("FirstDiff should locate the change, got: %v", got.FirstDiff)
	}
}

// TestNullVersusValueIsCaught: a value becoming NULL changes the per-column
// count as well, so both signals should fire.
func TestNullVersusValueIsCaught(t *testing.T) {
	nulled := [][3]any{base[0], base[1], {"003", "Lê Văn C", 5.0}}
	got := compare(t, base, nulled)
	if got.OK() {
		t.Fatal("a NULL becoming a value was not detected")
	}
	if !strings.Contains(strings.Join(got.Problems, ";"), "non-NULL count for toan") {
		t.Errorf("expected a per-column count difference, got %v", got.Problems)
	}
}

func TestRowCountDifferenceIsCaught(t *testing.T) {
	got := compare(t, base, base[:2])
	if got.OK() {
		t.Fatal("a missing row was not detected")
	}
	if !strings.Contains(strings.Join(got.Problems, ";"), "row count 3 vs 2") {
		t.Errorf("problems = %v", got.Problems)
	}
}

// TestTextShiftAcrossFieldsIsCaught guards the field separator. Hashing the
// values joined bare would make ("ab","c") and ("a","bc") identical, so a value
// shifted across a column boundary would slip through.
func TestTextShiftAcrossFieldsIsCaught(t *testing.T) {
	a := [][3]any{{"001", "ab", 1.0}}
	b := [][3]any{{"001", "a", 1.0}}
	// Same idea at the boundary between so_bao_danh and ho_ten.
	c := [][3]any{{"01", "23", 1.0}}
	d := [][3]any{{"012", "3", 1.0}}

	if compare(t, a, b).OK() {
		t.Error("a shortened text value was not detected")
	}
	if compare(t, c, d).OK() {
		t.Error("a value shifted across a field boundary was not detected")
	}
}

func TestMissingDatabaseIsAnError(t *testing.T) {
	dir := t.TempDir()
	makeDB(t, dir, "x", base)
	if _, err := Compare([]registry.Dataset{{ID: "x"}}, dir, t.TempDir()); err == nil {
		t.Fatal("comparing against a directory with no database must fail")
	}
}

func TestSer(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
	}{
		{nil, "\x00NULL"},
		{int64(7), "7"},
		{8.0, "8.0"}, // whole REALs get a fixed rendering
		{7.25, "7.25"},
		{"x", "x"},
		{[]byte("y"), "y"},
	} {
		if got := ser(tc.in); got != tc.want {
			t.Errorf("ser(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// A NULL must not collide with any real text value.
	if ser(nil) == ser("NULL") || ser(nil) == ser("") {
		t.Error("the NULL sentinel collides with a real value")
	}
}
