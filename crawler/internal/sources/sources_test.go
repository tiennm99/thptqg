package sources

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/crawler/internal/article"
)

// The fixtures are saved copies of the two source articles, kept so the link
// extraction and the naming rules can be exercised without the network.
//
// article-2017 is the live page. article-2016 came from the Internet Archive's
// copy of the site that first published that list (dtntbacgiang.edu.vn, which
// no longer resolves); the mirror the crawler actually reads carries the same
// article. Its hrefs are relative, so they resolve against whichever host the
// source names, and the filenames are identical either way.
func fixture(t *testing.T, id string) *gzip.Reader {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "article-"+id+".html.gz"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { gz.Close() })
	return gz
}

// resolveFixture runs a source's full pipeline over its saved article.
func resolveFixture(t *testing.T, src Source) []File {
	t.Helper()
	links, err := article.Extract(src.Article, fixture(t, src.ID), src.Exts...)
	if err != nil {
		t.Fatal(err)
	}
	files, err := src.Resolve(links)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestReproducesFilesOnDisk is the guard that matters.
//
// parser sorts its inputs and inserts last-wins, so filenames decide which
// row survives a duplicate exam number. If extraction or naming drifted, a
// re-crawl could rebuild a database with the same row count and different
// content, which the row-count guard in build-db.js would not catch.
//
// The committed data/<id>/ is the oracle: reading the source article and
// applying the source's naming rule must reproduce it exactly, both directions.
func TestReproducesFilesOnDisk(t *testing.T) {
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

			for _, f := range resolveFixture(t, src) {
				if !onDisk[f.Dest] {
					t.Errorf("crawler would write %q, which is not in %s — "+
						"a renamed input can change which duplicate row survives", f.Dest, dir)
				}
				delete(onDisk, f.Dest)
			}
			for name := range onDisk {
				t.Errorf("%s/%s exists but the article produces no link for it", dir, name)
			}
		})
	}
}

// TestResolvedURLsAreAbsolute: 2016's article uses relative hrefs, so a
// mis-resolved base would produce URLs that cannot be fetched at all.
func TestResolvedURLsAreAbsolute(t *testing.T) {
	for _, src := range All() {
		t.Run(src.ID, func(t *testing.T) {
			for _, f := range resolveFixture(t, src) {
				if !strings.HasPrefix(f.URL, "https://") {
					t.Errorf("%s: %q is not an absolute https URL", f.Dest, f.URL)
				}
			}
		})
	}
}

func TestExtensionsMatchWhatIsExpected(t *testing.T) {
	counts := map[string]map[string]int{}
	for _, src := range All() {
		c := map[string]int{}
		for _, f := range resolveFixture(t, src) {
			c[filepath.Ext(f.Dest)]++
		}
		counts[src.ID] = c
	}
	if got := counts["2016"]; got[".xls"] != 4 || got[".xlsx"] != 115 {
		t.Errorf("2016: %d .xls and %d .xlsx, want 4 and 115", got[".xls"], got[".xlsx"])
	}
	if got := counts["2017"]; got[".xls"] != 63 {
		t.Errorf("2017: %d .xls, want 63", got[".xls"])
	}
}

// TestResolveRejectsShortPage: a page that has changed shape must stop the
// crawl, not quietly produce a partial dataset.
func TestResolveRejectsShortPage(t *testing.T) {
	src := source2017
	_, err := src.Resolve([]article.Link{{URL: "https://x/a.xls", Text: "An Giang", File: "a.xls"}})
	if err == nil {
		t.Fatal("expected an error for a short link list")
	}
	if !strings.Contains(err.Error(), "expected 63") {
		t.Errorf("error should name the expected count, got: %v", err)
	}
}

// TestResolveRejectsCollidingNames: two links naming the same local file would
// cost the dataset a file with no error anywhere downstream.
func TestResolveRejectsCollidingNames(t *testing.T) {
	src := Source{
		ID: "t", WantFiles: 2,
		Dest: func(article.Link) (string, error) { return "same.xls", nil },
	}
	_, err := src.Resolve([]article.Link{
		{URL: "https://x/a.xls", File: "a.xls"},
		{URL: "https://x/b.xls", File: "b.xls"},
	})
	if err == nil || !strings.Contains(err.Error(), "same.xls") {
		t.Errorf("expected a collision error naming the file, got: %v", err)
	}
}

func TestSlug(t *testing.T) {
	// The article writes province names with diacritics; the committed
	// filenames are ASCII. These are the awkward ones.
	for in, want := range map[string]string{
		"An Giang":         "an-giang",
		"Bạc Liêu":         "bac-lieu",
		"Bắc Kạn":          "bac-kan",
		"Bà Rịa -Vũng Tàu": "ba-ria-vung-tau",
		"Thừa Thiên - Huế": "thua-thien-hue",
		"Đắk Lắk":          "dak-lak",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLookup(t *testing.T) {
	for _, s := range All() {
		got, err := Lookup(s.ID)
		if err != nil {
			t.Errorf("Lookup(%q): %v", s.ID, err)
		}
		if got.ID != s.ID || got.Article == "" || got.WantFiles == 0 {
			t.Errorf("Lookup(%q) returned an incomplete source: %+v", s.ID, got)
		}
	}
	if _, err := Lookup("nope"); err == nil {
		t.Error("Lookup of an unknown dataset must fail")
	}
}

// TestIDsAreDatasetIDs: the ID doubles as the directory under data/, so a typo
// would send a crawl into a directory the parser never reads.
func TestIDsAreDatasetIDs(t *testing.T) {
	for _, s := range All() {
		dir := filepath.Join("..", "..", "..", "data", s.ID)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("source %q has no dataset directory at %s", s.ID, dir)
		}
	}
}
