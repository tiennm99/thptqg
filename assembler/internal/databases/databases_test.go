package databases

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

// TestCompressRoundTripsAndRemovesTheSource: only the .gz may survive, so that
// shipping a 100+ MB uncompressed database is structurally impossible rather
// than left to a cleanup step.
func TestCompressRoundTripsAndRemovesTheSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "2016.db")
	body := []byte("pretend this is a SQLite file")
	if err := os.WriteFile(src, body, 0o644); err != nil {
		t.Fatal(err)
	}

	gzPath, size, err := compress(src)
	if err != nil {
		t.Fatal(err)
	}
	if gzPath != src+".gz" || size <= 0 {
		t.Fatalf("gzPath=%q size=%d", gzPath, size)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("the uncompressed database must not survive")
	}

	f, err := os.Open(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Errorf("round-trip gave %q", got)
	}
}

func TestCleanRemovesOnlyDroppedDatasets(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"2016.db.gz", "2017.db.gz",
		"2017-old.db.gz",  // dropped from the registry
		"2017-old2.db.gz", // dropped from the registry
		"2016.db-journal", // interrupted run
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p := Paths{OutDir: dir}
	keep := []registry.Dataset{{ID: "2016"}, {ID: "2017"}}
	if err := Clean(p, keep); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	slices.Sort(left)

	want := []string{"2016.db.gz", "2017.db.gz"}
	if !slices.Equal(left, want) {
		t.Errorf("left %v, want %v", left, want)
	}
}

// TestCleanToleratesAnAbsentStagingDirectory: a fresh checkout has never built
// anything, and that is not an error.
func TestCleanToleratesAnAbsentStagingDirectory(t *testing.T) {
	p := Paths{OutDir: filepath.Join(t.TempDir(), "never-created")}
	if err := Clean(p, nil); err != nil {
		t.Errorf("Clean on a missing directory should succeed, got %v", err)
	}
}

func TestDefaultPaths(t *testing.T) {
	p := DefaultPaths("/repo")
	if p.Parser != filepath.Join("/repo", "parser") {
		t.Errorf("Parser = %q", p.Parser)
	}
	// The staging directory must be the one Vite publishes, or the databases
	// never reach the site.
	if p.OutDir != filepath.Join("/repo", ".build", "public", "db") {
		t.Errorf("OutDir = %q", p.OutDir)
	}
}
