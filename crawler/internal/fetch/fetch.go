// Package fetch downloads a list of files concurrently, skipping any that are
// already on disk.
//
// It is deliberately source-agnostic: it knows nothing about provinces, exam
// years or spreadsheet formats. Each source in internal/sources produces a
// []Item and this package moves the bytes.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Item is one file to download.
type Item struct {
	// Name is the human label used in progress output, e.g. "An Giang".
	Name string
	URL  string
	// Path is the absolute destination, including filename.
	Path string
}

// Status is the outcome of one item.
type Status int

const (
	// StatusOK means the file was downloaded during this run.
	StatusOK Status = iota
	// StatusSkip means a non-empty file was already present.
	StatusSkip
	// StatusFail means the download did not complete; Result.Err says why.
	StatusFail
)

// Result records what happened to one item.
type Result struct {
	Item   Item
	Status Status
	Size   int64
	Err    error
}

// Options configures a run. The zero value is usable: Concurrency and Timeout
// fall back to defaults.
type Options struct {
	// Concurrency is the number of files downloaded at once.
	Concurrency int
	// Timeout bounds each individual request, headers and body together.
	// The original JS crawler had no timeout, so one stalled connection could
	// hang the whole run indefinitely.
	Timeout time.Duration
	// Headers are sent with every request. The CDN this was written against
	// rejects requests without a browser User-Agent and a matching Referer.
	Headers map[string]string
	// OnResult, if set, is called once per completed item. Calls are
	// serialised, so it does not need its own locking, but they arrive in
	// completion order rather than list order.
	OnResult func(done, total int, r Result)
}

const (
	defaultConcurrency = 6
	defaultTimeout     = 2 * time.Minute
)

// Run downloads every item, returning one Result each. It does not return an
// error for a failed download — that is reported per item — only for a problem
// that stops the run as a whole, such as an unusable output directory.
//
// Results come back in completion order, not input order.
func Run(ctx context.Context, items []Item, opts Options) ([]Result, error) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}

	// Every item's parent directory must exist before any worker starts, so a
	// missing output directory fails once here rather than N times in parallel.
	for _, it := range items {
		if err := os.MkdirAll(filepath.Dir(it.Path), 0o755); err != nil {
			return nil, fmt.Errorf("cannot create output directory: %w", err)
		}
	}

	client := &http.Client{Timeout: opts.Timeout}

	var (
		mu      sync.Mutex
		results = make([]Result, 0, len(items))
		next    int
	)

	work := make(chan Item)
	var wg sync.WaitGroup
	for range opts.Concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range work {
				r := download(ctx, client, it, opts.Headers)
				mu.Lock()
				results = append(results, r)
				next++
				if opts.OnResult != nil {
					opts.OnResult(next, len(items), r)
				}
				mu.Unlock()
			}
		}()
	}

	for _, it := range items {
		select {
		case work <- it:
		case <-ctx.Done():
			close(work)
			wg.Wait()
			return results, ctx.Err()
		}
	}
	close(work)
	wg.Wait()

	return results, nil
}

// download fetches one item, or reports that it was already present.
//
// The body is streamed to a .part file and renamed into place only once it is
// complete. Without that, an interrupted run leaves a truncated file at the
// final path — and because the skip check only tests for a non-empty file,
// every later run would skip it and the corruption would persist silently.
func download(ctx context.Context, client *http.Client, it Item, headers map[string]string) Result {
	if st, err := os.Stat(it.Path); err == nil && st.Size() > 0 {
		return Result{Item: it, Status: StatusSkip, Size: st.Size()}
	}

	fail := func(err error) Result {
		return Result{Item: it, Status: StatusFail, Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, it.URL, nil)
	if err != nil {
		return fail(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fail(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	part := it.Path + ".part"
	f, err := os.Create(part)
	if err != nil {
		return fail(err)
	}
	n, err := io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(part)
		return fail(err)
	}
	if n == 0 {
		os.Remove(part)
		return fail(fmt.Errorf("empty response body"))
	}
	if err := os.Rename(part, it.Path); err != nil {
		os.Remove(part)
		return fail(err)
	}

	return Result{Item: it, Status: StatusOK, Size: n}
}

// Tally counts results by status.
func Tally(results []Result) (ok, skip int, failed []Result) {
	for _, r := range results {
		switch r.Status {
		case StatusOK:
			ok++
		case StatusSkip:
			skip++
		case StatusFail:
			failed = append(failed, r)
		}
	}
	return ok, skip, failed
}
