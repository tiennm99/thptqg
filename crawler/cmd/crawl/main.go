// Command crawl downloads a dataset's source spreadsheets into data/<id>/.
//
// The argument is the dataset id, the same one used by go-parser's configs and
// the published site paths.
//
//	crawl 2017                      # from the baotintuc.vn CDN
//	crawl 2016                      # source not yet configured
//	crawl 2017 --list               # show what would be downloaded
//
// Runs are idempotent: a file already present and non-empty is skipped, so an
// interrupted crawl can simply be re-run.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tiennm99/thptqg/crawler/internal/fetch"
	"github.com/tiennm99/thptqg/crawler/internal/sources"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "crawl: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, "usage: crawl <dataset> [flags]\n\nDatasets:\n")
	for _, s := range sources.All() {
		fmt.Fprintf(os.Stderr, "  %-6s %s\n", s.ID, s.Summary)
	}
	fmt.Fprint(os.Stderr, "\nFlags:\n")
	fmt.Fprint(os.Stderr, "  --out string          output directory (default ../data/<dataset>)\n")
	fmt.Fprint(os.Stderr, "  --concurrency int     parallel downloads (default 6)\n")
	fmt.Fprint(os.Stderr, "  --timeout duration    per-file timeout (default 2m)\n")
	fmt.Fprint(os.Stderr, "  --list                print the file list and exit\n")
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		usage()
		if len(args) == 0 {
			return fmt.Errorf("no dataset given")
		}
		return nil
	}

	src, err := sources.Lookup(args[0])
	if err != nil {
		usage()
		return err
	}

	fs := flag.NewFlagSet(src.ID, flag.ContinueOnError)
	// The default is relative to the crawler module directory, which is where
	// both `go -C crawler run ./cmd/crawl` and a manual `cd crawler` land.
	out := fs.String("out", filepath.Join("..", "data", src.ID), "output directory")
	concurrency := fs.Int("concurrency", 6, "parallel downloads")
	timeout := fs.Duration("timeout", 2*time.Minute, "per-file timeout")
	list := fs.Bool("list", false, "print the file list and exit")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	files, err := src.Files()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("source %q produced no files", src.ID)
	}

	outDir, err := filepath.Abs(*out)
	if err != nil {
		return err
	}

	items := make([]fetch.Item, 0, len(files))
	for _, f := range files {
		items = append(items, fetch.Item{
			Name: f.Name,
			URL:  f.URL,
			Path: filepath.Join(outDir, f.Dest),
		})
	}

	if *list {
		for _, it := range items {
			fmt.Printf("%s\t%s\n", filepath.Base(it.Path), it.URL)
		}
		return nil
	}

	// Ctrl-C cancels in-flight requests; each worker deletes its partial file
	// on the way out, so an interrupted run leaves nothing half-written.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Downloading %d files to %s...\n", len(items), outDir)

	results, runErr := fetch.Run(ctx, items, fetch.Options{
		Concurrency: *concurrency,
		Timeout:     *timeout,
		Headers:     src.Headers,
		OnResult:    printResult,
	})
	if runErr != nil {
		return runErr
	}

	ok, skip, failed := fetch.Tally(results)
	fmt.Printf("\nDone. ok=%d skip=%d fail=%d\n", ok, skip, len(failed))
	if len(failed) > 0 {
		for _, r := range failed {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", r.Item.Name, r.Err)
		}
		return fmt.Errorf("%d file(s) failed", len(failed))
	}
	return nil
}

func printResult(done, total int, r fetch.Result) {
	tag := map[fetch.Status]string{
		fetch.StatusOK:   "✓",
		fetch.StatusSkip: "·",
		fetch.StatusFail: "✗",
	}[r.Status]

	detail := ""
	switch r.Status {
	case fetch.StatusFail:
		detail = r.Err.Error()
	default:
		detail = fmt.Sprintf("%.0f KB", float64(r.Size)/1024)
	}
	fmt.Printf("  %s [%d/%d] %-20s %s\n", tag, done, total, r.Item.Name, detail)
}
