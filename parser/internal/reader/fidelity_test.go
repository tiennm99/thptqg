package reader_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/parser/internal/reader"
)

// TestReaderFidelity asserts the reader reproduces the reference rendering
// byte-for-byte on every real input file.
//
// The oracle is parser/testdata/reader-fidelity-hashes.tsv: one SHA-256 per
// file over a canonical cell dump. The dumps are real student names and
// birthdates, so only the hashes are committed. The manifest is FROZEN — it is
// the only guard on reader output, so a mismatch is a reader bug until proven
// otherwise, never a cue to regenerate the hashes.
//
// The canonical form carries geometry and rendered cell values only. Whether a
// cell was absent or an empty string is excluded on purpose: both render "" and
// both count as blank in IsAllBlank and transform, so the distinction cannot
// reach the database.
//
// Runs by default so CI and `go test ./...` keep the full guarantee; skipped
// under -short, which is how to iterate without paying ~77s to re-read 418 MB.
func TestReaderFidelity(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: skipping the full-corpus sweep")
	}
	root := repoRoot(t)
	manifest := filepath.Join(root, "parser", "testdata", "reader-fidelity-hashes.tsv")

	f, err := os.Open(manifest)
	if err != nil {
		t.Fatalf("open manifest: %v", err)
	}
	defer f.Close()

	var checked int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rel, want, ok := strings.Cut(line, "\t")
		if !ok {
			t.Fatalf("malformed manifest line: %q", line)
		}
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("input data not present (%s); skipping fidelity suite", rel)
		}

		checked++
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			got, err := canonicalHash(path)
			if err != nil {
				t.Fatalf("hash %s: %v", rel, err)
			}
			if got != want {
				t.Errorf("cell dump diverges from the frozen oracle\n  want %s\n  got  %s", want, got)
			}
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if checked == 0 {
		t.Fatal("manifest contained no entries")
	}
}

// canonicalHash renders one file in the canonical form and hashes it. The byte
// format is fixed: any change to it invalidates every hash in the manifest.
func canonicalHash(path string) (string, error) {
	wb, err := reader.Open(path)
	if err != nil {
		return "", err
	}
	defer wb.Close()

	h := sha256.New()
	sheets := wb.Sheets()
	fmt.Fprintf(h, "SHEETCOUNT\t%d\n", len(sheets))
	for _, sh := range sheets {
		fmt.Fprintf(h, "SHEET\t%d\t%s\t%d\t%d\n", sh.Index, escape(sh.Name), sh.Height, sh.Width)
		err := wb.EachRow(sh.Index, func(s reader.Sheet, rowIdx int, row []reader.Cell) error {
			fmt.Fprintf(h, "ROW\t%d\t%d\t%d\n", s.Index, rowIdx, len(row))
			for c, cell := range row {
				fmt.Fprintf(h, "CELL\t%d\t%d\t%d\t%s\n", s.Index, rowIdx, c, escape(cell.Str))
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func escape(s string) string {
	return strings.NewReplacer("\\", `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`).Replace(s)
}

// repoRoot walks up from the test's working directory to the directory holding
// the data/ corpus.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "data")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate repo root (no data/ directory found)")
	return ""
}
