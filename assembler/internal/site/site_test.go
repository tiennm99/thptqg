package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

var datasets = []registry.Dataset{{ID: "2016"}, {ID: "2017"}}

// fakeBuild stands in for a Vite build: an index.html, an asset, and whatever
// databases the caller wants staged.
func fakeBuild(t *testing.T, dbs ...string) Paths {
	t.Helper()
	root := t.TempDir()
	dist := filepath.Join(root, "web", "dist")
	write(t, filepath.Join(dist, "index.html"), "<html>app</html>")
	write(t, filepath.Join(dist, "assets", "index.js"), "console.log(1)")
	for _, name := range dbs {
		write(t, filepath.Join(dist, "db", name), "gzipped-bytes")
	}
	return Paths{Root: root, Web: filepath.Join(root, "web"), Dist: dist, Site: filepath.Join(root, "_site")}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleProducesAPageForEveryDataset(t *testing.T) {
	p := fakeBuild(t, "2016.db.gz", "2017.db.gz")
	if err := Assemble(p, datasets); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"index.html",
		"404.html",
		filepath.Join("2016", "index.html"),
		filepath.Join("2017", "index.html"),
		filepath.Join("assets", "index.js"),
		filepath.Join("db", "2016.db.gz"),
	} {
		if _, err := os.Stat(filepath.Join(p.Site, want)); err != nil {
			t.Errorf("missing from the artifact: %s", want)
		}
	}
}

// TestMissingDatabaseFailsTheBuild is the guard that closes the widest hole.
//
// Without it the site assembles happily with an empty db/ directory: every page
// renders, every query 404s, and CI stays green. The row-count and size guards
// cannot catch this — they only run when a database was built at all.
func TestMissingDatabaseFailsTheBuild(t *testing.T) {
	p := fakeBuild(t, "2016.db.gz") // 2017 never built
	err := Assemble(p, datasets)
	if err == nil {
		t.Fatal("expected an error when a database is missing")
	}
	if !strings.Contains(err.Error(), "2017.db.gz") {
		t.Errorf("the error should name the missing database, got: %v", err)
	}
}

// TestEmptyDatabaseFailsTheBuild: a zero-byte file satisfies "exists" but is
// not a database.
func TestEmptyDatabaseFailsTheBuild(t *testing.T) {
	p := fakeBuild(t, "2016.db.gz", "2017.db.gz")
	write(t, filepath.Join(p.Dist, "db", "2017.db.gz"), "")
	if err := Assemble(p, datasets); err == nil {
		t.Fatal("expected an error for a zero-byte database")
	}
}

// TestRawDatabaseFailsTheBuild: the compression step deletes its source, so a
// raw .db here means an interrupted run left one behind — and it is 100+ MB.
func TestRawDatabaseFailsTheBuild(t *testing.T) {
	for _, name := range []string{"2016.db", "2016.db-journal", "2016.db-wal", "2016.db-shm"} {
		t.Run(name, func(t *testing.T) {
			p := fakeBuild(t, "2016.db.gz", "2017.db.gz")
			write(t, filepath.Join(p.Dist, "db", name), "raw sqlite")
			err := Assemble(p, datasets)
			if err == nil {
				t.Fatalf("expected an error for %s", name)
			}
			if !strings.Contains(err.Error(), "uncompressed") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestGzipIsNotMistakenForRaw: the reject pattern is anchored, so a .db.gz must
// pass. Getting this wrong would fail every build.
func TestGzipIsNotMistakenForRaw(t *testing.T) {
	if rawDatabase.MatchString("2016.db.gz") {
		t.Error("a .db.gz must not be treated as an uncompressed database")
	}
	for _, name := range []string{"2016.db", "x.db-journal", "x.db-wal", "x.db-shm"} {
		if !rawDatabase.MatchString(name) {
			t.Errorf("%s should be treated as an uncompressed artifact", name)
		}
	}
}

func TestAssembleRejectsAMissingBuild(t *testing.T) {
	root := t.TempDir()
	p := Paths{Root: root, Web: root, Dist: filepath.Join(root, "dist"), Site: filepath.Join(root, "_site")}
	if err := Assemble(p, datasets); err == nil {
		t.Fatal("expected an error when there is no Vite build")
	}
}

// TestAssembleIsIdempotent: the site directory is rebuilt from scratch, so a
// previous run's leftovers cannot survive into the artifact.
func TestAssembleIsIdempotent(t *testing.T) {
	p := fakeBuild(t, "2016.db.gz", "2017.db.gz")
	if err := Assemble(p, datasets); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(p.Site, "2015", "index.html")
	write(t, stale, "old dataset")
	if err := Assemble(p, datasets); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("a directory from a previous run survived into the new artifact")
	}
}
