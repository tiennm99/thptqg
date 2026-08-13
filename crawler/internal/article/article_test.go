package article

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const page = "https://example.test/news/scores.html"

func extract(t *testing.T, body string, exts ...string) []Link {
	t.Helper()
	links, err := Extract(page, strings.NewReader(body), exts...)
	if err != nil {
		t.Fatal(err)
	}
	return links
}

// TestResolvesRelativeHrefs is the behaviour the 2016 source depends on: its
// article links files as /upload/..., so they only become fetchable once
// resolved against the page they were found on.
func TestResolvesRelativeHrefs(t *testing.T) {
	got := extract(t, `
		<a href="/upload/s/a.xlsx">root-relative</a>
		<a href="sub/b.xlsx">page-relative</a>
		<a href="https://cdn.other/c.xlsx">already absolute</a>
		<a href="//cdn.other/d.xlsx">protocol-relative</a>
	`, ".xlsx")

	want := []string{
		"https://example.test/upload/s/a.xlsx",
		"https://example.test/news/sub/b.xlsx",
		"https://cdn.other/c.xlsx",
		"https://cdn.other/d.xlsx",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d links, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].URL != w {
			t.Errorf("link %d = %q, want %q", i, got[i].URL, w)
		}
	}
}

func TestFiltersByExtension(t *testing.T) {
	got := extract(t, `
		<a href="/a.xls">keep</a>
		<a href="/b.xlsx">keep</a>
		<a href="/c.pdf">drop</a>
		<a href="/d.html">drop</a>
		<a href="/e.XLSX">keep, case-insensitive</a>
		<a>no href at all</a>
	`, ".xls", ".xlsx")

	if len(got) != 3 {
		t.Fatalf("got %d links, want 3: %+v", len(got), got)
	}
	// ".xls" must not swallow ".xlsx" or vice versa when only one is asked for.
	only := extract(t, `<a href="/a.xls">x</a><a href="/b.xlsx">y</a>`, ".xlsx")
	if len(only) != 1 || only[0].File != "b.xlsx" {
		t.Errorf("extension filter is too loose: %+v", only)
	}
}

// TestFileIgnoresQueryString: a query string is not part of the filename, and
// writing one to disk would produce a name go-parser never sees.
func TestFileIgnoresQueryString(t *testing.T) {
	got := extract(t, `<a href="/d/report.xlsx?v=2&t=3">x</a>`, ".xlsx")
	if len(got) != 1 {
		t.Fatalf("got %d links", len(got))
	}
	if got[0].File != "report.xlsx" {
		t.Errorf("File = %q, want report.xlsx", got[0].File)
	}
	if !strings.Contains(got[0].URL, "v=2") {
		t.Errorf("the query string must survive in the URL: %q", got[0].URL)
	}
}

// TestTextCollectsNestedMarkup: both real articles wrap the label in styling
// tags, so the anchor text is never a single child node. 2017 names its files
// from this text.
func TestTextCollectsNestedMarkup(t *testing.T) {
	got := extract(t, `<a href="/a.xls"><span style="x"><b>Bà Rịa</b> -Vũng&nbsp;Tàu</span></a>`, ".xls")
	if len(got) != 1 {
		t.Fatalf("got %d links", len(got))
	}
	if got[0].Text != "Bà Rịa -Vũng Tàu" {
		t.Errorf("Text = %q, want %q", got[0].Text, "Bà Rịa -Vũng Tàu")
	}
}

func TestEntitiesInHrefAreDecoded(t *testing.T) {
	got := extract(t, `<a href="/d/a.xlsx?x=1&amp;y=2">x</a>`, ".xlsx")
	if len(got) != 1 || !strings.Contains(got[0].URL, "x=1&y=2") {
		t.Errorf("href entity not decoded: %+v", got)
	}
}

// TestMalformedHTMLStillParses: these are hand-edited CMS pages, and the parser
// must not give up on unclosed tags.
func TestMalformedHTMLStillParses(t *testing.T) {
	got := extract(t, `<p><b>list<a href="/a.xls">one<a href="/b.xls">two`, ".xls")
	if len(got) != 2 {
		t.Errorf("got %d links, want 2: %+v", len(got), got)
	}
}

func TestExtractRejectsBadPageURL(t *testing.T) {
	if _, err := Extract("://nonsense", strings.NewReader("<a href=/a.xls>x</a>"), ".xls"); err == nil {
		t.Error("expected an error for an unparseable page URL")
	}
}

func TestFetchSendsHeadersAndParses(t *testing.T) {
	var ua, ref string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua, ref = r.Header.Get("User-Agent"), r.Header.Get("Referer")
		w.Write([]byte(`<a href="/x/a.xls">An Giang</a>`))
	}))
	defer srv.Close()

	links, err := Fetch(t.Context(), srv.Client(), srv.URL,
		map[string]string{"User-Agent": "test-agent", "Referer": "https://ref.test/"}, ".xls")
	if err != nil {
		t.Fatal(err)
	}
	if ua != "test-agent" || ref != "https://ref.test/" {
		t.Errorf("headers not sent: ua=%q referer=%q", ua, ref)
	}
	if len(links) != 1 || links[0].Text != "An Giang" || links[0].File != "a.xls" {
		t.Errorf("unexpected links: %+v", links)
	}
}

// TestFetchFailsLoudly: a CMS that answers 404 or 403 with an HTML error page
// would otherwise yield zero links and read as "nothing to download".
func TestFetchFailsLoudly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "<html>not found</html>", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := Fetch(t.Context(), srv.Client(), srv.URL, nil, ".xls"); err == nil {
		t.Fatal("expected an error for a non-200 response")
	} else if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should name the status, got: %v", err)
	}
}
