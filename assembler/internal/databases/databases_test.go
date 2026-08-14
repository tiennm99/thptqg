package databases

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

func TestCleanRemovesOnlyDroppedDatasets(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"2016.sqlite3", "2017.sqlite3",
		"2017-old.sqlite3",  // dropped from the registry
		"2017-old2.sqlite3", // dropped from the registry
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

	want := []string{"2016.sqlite3", "2017.sqlite3"}
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
