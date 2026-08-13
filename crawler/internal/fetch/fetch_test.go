package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s.URL
}

func TestDownloadsAndSkips(t *testing.T) {
	var hits int
	url := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Write([]byte("payload"))
	})

	dir := t.TempDir()
	items := []Item{{Name: "one", URL: url, Path: filepath.Join(dir, "one.xls")}}

	results, err := Run(context.Background(), items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != StatusOK {
		t.Fatalf("status = %v, err = %v", results[0].Status, results[0].Err)
	}
	if got, _ := os.ReadFile(items[0].Path); string(got) != "payload" {
		t.Errorf("content = %q", got)
	}

	// Second run must not re-fetch.
	results, err = Run(context.Background(), items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != StatusSkip {
		t.Errorf("status = %v, want StatusSkip", results[0].Status)
	}
	if hits != 1 {
		t.Errorf("server hit %d times, want 1", hits)
	}
}

func TestNon200Fails(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "missing.xls")
	results, err := Run(context.Background(), []Item{{Name: "x", URL: url, Path: path}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != StatusFail {
		t.Fatalf("status = %v, want StatusFail", results[0].Status)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a failed download must leave no file at the destination")
	}
	assertNoPartFiles(t, dir)
}

// TestFailureLeavesNoPartial is the reason downloads land on a .part file
// first. Writing straight to the destination would leave a truncated file
// there, and the skip check — which only tests for a non-empty file — would
// skip it on every later run, so the corruption would never be re-fetched.
func TestFailureLeavesNoPartial(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("short"))
		// Closing early makes the body read fail mid-copy.
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler)
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "truncated.xls")
	results, err := Run(context.Background(), []Item{{Name: "x", URL: url, Path: path}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != StatusFail {
		t.Fatalf("status = %v, want StatusFail", results[0].Status)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a truncated download must not be left at the destination")
	}
	assertNoPartFiles(t, dir)
}

func TestEmptyBodyFails(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, _ *http.Request) {})

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.xls")
	results, err := Run(context.Background(), []Item{{Name: "x", URL: url, Path: path}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != StatusFail {
		t.Errorf("an empty body must fail, got %v", results[0].Status)
	}
	assertNoPartFiles(t, dir)
}

func TestHeadersAreSent(t *testing.T) {
	var gotUA, gotRef string
	url := serve(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotRef = r.Header.Get("User-Agent"), r.Header.Get("Referer")
		w.Write([]byte("ok"))
	})

	_, err := Run(context.Background(),
		[]Item{{Name: "x", URL: url, Path: filepath.Join(t.TempDir(), "x.xls")}},
		Options{Headers: map[string]string{"User-Agent": "test-agent", "Referer": "https://example.test/"}})
	if err != nil {
		t.Fatal(err)
	}
	if gotUA != "test-agent" || gotRef != "https://example.test/" {
		t.Errorf("headers not sent: ua=%q referer=%q", gotUA, gotRef)
	}
}

func TestAllItemsRun(t *testing.T) {
	url := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("x"))
	})

	dir := t.TempDir()
	var items []Item
	for i := range 20 {
		items = append(items, Item{
			Name: "f",
			URL:  url,
			Path: filepath.Join(dir, "sub", string(rune('a'+i))+".xls"),
		})
	}

	results, err := Run(context.Background(), items, Options{Concurrency: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(items) {
		t.Fatalf("got %d results, want %d", len(results), len(items))
	}
	ok, _, failed := Tally(results)
	if ok != len(items) {
		t.Errorf("ok = %d, want %d (failures: %v)", ok, len(items), failed)
	}
}

func TestTally(t *testing.T) {
	ok, skip, failed := Tally([]Result{
		{Status: StatusOK}, {Status: StatusOK}, {Status: StatusSkip}, {Status: StatusFail},
	})
	if ok != 2 || skip != 1 || len(failed) != 1 {
		t.Errorf("ok=%d skip=%d fail=%d", ok, skip, len(failed))
	}
}

func assertNoPartFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".part" {
			t.Errorf("leftover partial file: %s", e.Name())
		}
	}
}
