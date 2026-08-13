package sources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"An Giang":          "an-giang",
		"Ba Ria - Vung Tau": "ba-ria-vung-tau",
		"Ho Chi Minh":       "ho-chi-minh",
		"Thua Thien Hue":    "thua-thien-hue",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSource2017IsComplete(t *testing.T) {
	files, err := source2017.Files()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 63 {
		t.Errorf("got %d provinces, want 63", len(files))
	}

	// A duplicate Dest would silently overwrite one province with another and
	// leave the dataset one file short, with no error anywhere.
	seenDest := map[string]string{}
	seenURL := map[string]string{}
	for _, f := range files {
		if prev, dup := seenDest[f.Dest]; dup {
			t.Errorf("%q and %q both write to %s", prev, f.Name, f.Dest)
		}
		seenDest[f.Dest] = f.Name
		if prev, dup := seenURL[f.URL]; dup {
			t.Errorf("%q and %q share a URL: %s", prev, f.Name, f.URL)
		}
		seenURL[f.URL] = f.Name

		if !strings.HasPrefix(f.URL, "https://") {
			t.Errorf("%s: URL is not https: %s", f.Name, f.URL)
		}
		if !strings.HasSuffix(f.Dest, ".xls") {
			t.Errorf("%s: Dest %q does not end in .xls", f.Name, f.Dest)
		}
	}
}

// TestMatchesFilesOnDisk is the guard that matters.
//
// go-parser sorts its inputs and inserts last-wins, so filenames decide which
// row survives a duplicate exam number. If a re-crawl started naming files
// differently, the rebuilt database could differ in content while still passing
// the row-count guard in build-db.js. The committed data/<id>/ is the oracle:
// whatever a source would write must match it exactly, in both directions.
//
// For 2016 this is also the only evidence that the recovered link list is the
// right one — the host that served those files is gone, so the list cannot be
// checked by fetching it.
func TestMatchesFilesOnDisk(t *testing.T) {
	for _, src := range All() {
		t.Run(src.ID, func(t *testing.T) {
			dir := filepath.Join("..", "..", "..", "data", src.ID)
			entries, err := os.ReadDir(dir)
			if os.IsNotExist(err) {
				t.Skipf("%s not present", dir)
			}
			if err != nil {
				t.Fatal(err)
			}

			onDisk := map[string]bool{}
			for _, e := range entries {
				if !e.IsDir() {
					onDisk[e.Name()] = true
				}
			}

			files, err := src.Files()
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range files {
				if !onDisk[f.Dest] {
					t.Errorf("crawler would write %q, which is not in %s — "+
						"a renamed input can change which duplicate row survives", f.Dest, dir)
				}
				delete(onDisk, f.Dest)
			}
			for name := range onDisk {
				t.Errorf("%s/%s exists but no source produces it", dir, name)
			}
		})
	}
}

func TestLookup(t *testing.T) {
	for _, s := range All() {
		got, err := Lookup(s.ID)
		if err != nil {
			t.Errorf("Lookup(%q): %v", s.ID, err)
		}
		if got.ID != s.ID || got.Summary == "" {
			t.Errorf("Lookup(%q) returned %+v", s.ID, got)
		}
	}
	if _, err := Lookup("nope"); err == nil {
		t.Error("Lookup of an unknown source must fail")
	}
}

// TestIDsAreDatasetIDs: the ID doubles as the directory under data/, so a typo
// would send a crawl into a new directory the parser never reads.
func TestIDsAreDatasetIDs(t *testing.T) {
	for _, s := range All() {
		dir := filepath.Join("..", "..", "..", "data", s.ID)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("source %q has no dataset directory at %s", s.ID, dir)
		}
	}
}

// TestSource2016IsComplete: 4 .xls + 115 .xlsx, matching what the source
// article published and what data/2016/ holds.
func TestSource2016IsComplete(t *testing.T) {
	files, err := source2016.Files()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 119 {
		t.Errorf("got %d clusters, want 119", len(files))
	}
	var xls, xlsx int
	for _, f := range files {
		switch filepath.Ext(f.Dest) {
		case ".xls":
			xls++
		case ".xlsx":
			xlsx++
		default:
			t.Errorf("%s: unexpected extension in %q", f.Name, f.Dest)
		}
		if !strings.HasPrefix(f.URL, baseURL+uploadPath) {
			t.Errorf("%s: URL does not use the configured base: %s", f.Name, f.URL)
		}
	}
	if xls != 4 || xlsx != 115 {
		t.Errorf("got %d .xls and %d .xlsx, want 4 and 115", xls, xlsx)
	}
}
