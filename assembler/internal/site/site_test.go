package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

var datasets = []registry.Dataset{{ID: "2016"}, {ID: "2017"}}

// fakeBuild stands in for a web build: the prerendered hub and one page per
// dataset, an asset, and whatever databases the caller wants staged.
func fakeBuild(t *testing.T, dbs ...string) Paths {
	t.Helper()
	root := t.TempDir()
	dist := filepath.Join(root, "web", "dist")
	write(t, filepath.Join(dist, "index.html"), "<html>hub</html>")
	for _, d := range datasets {
		write(t, filepath.Join(dist, d.ID, "index.html"), "<html>"+d.ID+"</html>")
	}
	write(t, filepath.Join(dist, "_app", "immutable", "entry.js"), "console.log(1)")
	for _, name := range dbs {
		write(t, filepath.Join(dist, "db", name), "sqlite-bytes")
	}
	return Paths{Web: filepath.Join(root, "web"), Dist: dist, Site: filepath.Join(root, "_site")}
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
	p := fakeBuild(t, "2016.sqlite3", "2017.sqlite3")
	if err := Assemble(p, datasets); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"index.html",
		"404.html",
		filepath.Join("2016", "index.html"),
		filepath.Join("2017", "index.html"),
		filepath.Join("_app", "immutable", "entry.js"),
		filepath.Join("db", "2016.sqlite3"),
	} {
		if _, err := os.Stat(filepath.Join(p.Site, want)); err != nil {
			t.Errorf("missing from the artifact: %s", want)
		}
	}
}

// TestMissingDatasetPageFailsTheBuild: the pages come from the web build's
// entry generator, which reads the same registry this does. If the two fall out
// of step, that dataset's URL 404s — so the build stops instead.
func TestMissingDatasetPageFailsTheBuild(t *testing.T) {
	p := fakeBuild(t, "2016.sqlite3", "2017.sqlite3")
	if err := os.RemoveAll(filepath.Join(p.Dist, "2017")); err != nil {
		t.Fatal(err)
	}
	err := Assemble(p, datasets)
	if err == nil {
		t.Fatal("expected an error when a dataset page was not prerendered")
	}
	if !strings.Contains(err.Error(), "2017") {
		t.Errorf("the error should name the missing page, got: %v", err)
	}
}

// TestMissingDatabaseFailsTheBuild is the guard that closes the widest hole.
//
// Without it the site assembles happily with an empty db/ directory: every page
// renders, every query 404s, and CI stays green. The row-count and size guards
// cannot catch this — they only run when a database was built at all.
func TestMissingDatabaseFailsTheBuild(t *testing.T) {
	p := fakeBuild(t, "2016.sqlite3") // 2017 never built
	err := Assemble(p, datasets)
	if err == nil {
		t.Fatal("expected an error when a database is missing")
	}
	if !strings.Contains(err.Error(), "2017.sqlite3") {
		t.Errorf("the error should name the missing database, got: %v", err)
	}
}

// TestEmptyDatabaseFailsTheBuild: a zero-byte file satisfies "exists" but is
// not a database.
func TestEmptyDatabaseFailsTheBuild(t *testing.T) {
	p := fakeBuild(t, "2016.sqlite3", "2017.sqlite3")
	write(t, filepath.Join(p.Dist, "db", "2017.sqlite3"), "")
	if err := Assemble(p, datasets); err == nil {
		t.Fatal("expected an error for a zero-byte database")
	}
}

// TestStrayArtifactFailsTheBuild: a journal means an interrupted run, a .db or
// a .sqlite30 means a name from an earlier design that no client asks for now,
// a .gz means a database left compressed — and each is 100+ MB.
func TestStrayArtifactFailsTheBuild(t *testing.T) {
	for _, name := range []string{
		"2016.db", "2016.sqlite30",
		"2016.sqlite3-journal", "2016.sqlite3-wal", "2016.sqlite3-shm", "2016.sqlite3.gz",
	} {
		t.Run(name, func(t *testing.T) {
			p := fakeBuild(t, "2016.sqlite3", "2017.sqlite3")
			write(t, filepath.Join(p.Dist, "db", name), "raw sqlite")
			err := Assemble(p, datasets)
			if err == nil {
				t.Fatalf("expected an error for %s", name)
			}
			if !strings.Contains(err.Error(), "stray") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestPublishedDatabaseIsNotMistakenForStray: the pattern must pass the one
// file the site is built to serve. Getting this wrong would fail every build.
func TestPublishedDatabaseIsNotMistakenForStray(t *testing.T) {
	if strayArtifact.MatchString("2016.sqlite3") {
		t.Error("the published database must not be treated as a stray artifact")
	}
	for _, name := range []string{
		"2016.db", "x.sqlite30", "x.sqlite3-journal", "x.sqlite3-wal", "x.sqlite3-shm",
		"x.sqlite30-journal", "x.sqlite3.gz", "x.db.gz",
	} {
		if !strayArtifact.MatchString(name) {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestAssembleRejectsAMissingBuild(t *testing.T) {
	root := t.TempDir()
	p := Paths{Web: root, Dist: filepath.Join(root, "dist"), Site: filepath.Join(root, "_site")}
	if err := Assemble(p, datasets); err == nil {
		t.Fatal("expected an error when there is no web build")
	}
}

// TestAssembleIsIdempotent: the site directory is rebuilt from scratch, so a
// previous run's leftovers cannot survive into the artifact.
func TestAssembleIsIdempotent(t *testing.T) {
	p := fakeBuild(t, "2016.sqlite3", "2017.sqlite3")
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
