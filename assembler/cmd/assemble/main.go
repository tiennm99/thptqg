// Command assemble turns source data and the web app into the directory
// GitHub Pages publishes.
//
//	assemble            # databases, then the site
//	assemble db         # databases only (add ids to limit: assemble db 2017)
//	assemble site       # web build and _site only, reusing staged databases
//	assemble verify A B # compare two sets of built databases field by field
//
// It sequences the other stages rather than doing their work: the parser reads
// spreadsheets, Vite bundles the app, and this decides what runs, checks what
// came out, and refuses to publish anything that looks wrong.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tiennm99/thptqg/assembler/internal/databases"
	"github.com/tiennm99/thptqg/assembler/internal/registry"
	"github.com/tiennm99/thptqg/assembler/internal/site"
	"github.com/tiennm99/thptqg/assembler/internal/verify"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "assemble: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: assemble [step] [dataset...]

Steps:
  (none)        databases, then the site
  db            build, verify and compress the databases
  site          build the web app and assemble _site
  verify A B    compare two directories of built databases, field by field

Naming datasets limits the db step to those; the site step always covers all of
them, since a partial site would publish links to databases it did not build.

verify takes two directories holding <id>.db.gz (or <id>.db) — for example a
copy of the previous .build/public/db against a fresh build. Relative paths
resolve against the repository root, not the working directory. It reports row
counts, per-column non-NULL counts, a full-table hash and the first differing
rows.

Nothing else checks database content: the reader oracle covers reading and the
row-count guard covers how many rows, but a change in transform or writer logic
can alter what is in them while both still pass.
`)
}

func run(args []string) error {
	step := ""
	if len(args) > 0 {
		switch args[0] {
		case "db", "site", "verify":
			step, args = args[0], args[1:]
		case "-h", "--help":
			usage()
			return nil
		default:
			// Bare dataset ids are a natural thing to type; treat them as the
			// db step rather than rejecting them.
			step = "db"
		}
	}

	root, err := repoRoot()
	if err != nil {
		return err
	}

	all, err := registry.Load(root)
	if err != nil {
		return err
	}

	if step == "verify" {
		if len(args) != 2 {
			usage()
			return fmt.Errorf("verify needs two directories")
		}
		// Relative paths resolve against the repository root, not the working
		// directory. The documented invocation is `go -C assembler run ...`,
		// and -C moves the process into assembler/ before main starts — so a
		// bare `.build/public/db` would otherwise look for it in there and
		// report a missing database that is sitting in plain sight.
		return runVerify(all, atRoot(root, args[0]), atRoot(root, args[1]))
	}

	if step == "" || step == "db" {
		selected, err := registry.Select(all, args)
		if err != nil {
			return err
		}
		if err := buildDatabases(root, all, selected); err != nil {
			return err
		}
	}

	if step == "" || step == "site" {
		sp := site.DefaultPaths(root)
		if err := site.BuildWeb(sp); err != nil {
			return err
		}
		if err := site.Assemble(sp, all); err != nil {
			return err
		}
	}
	return nil
}

// atRoot resolves a relative path against the repository root.
func atRoot(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// runVerify compares two sets of built databases and fails loudly on any
// difference. A dataset missing from either side is an error rather than a
// skip: silently comparing one of two datasets is how a gate passes without
// proving anything.
func runVerify(all []registry.Dataset, dirA, dirB string) error {
	results, err := verify.Compare(all, dirA, dirB)
	if err != nil {
		return err
	}

	bad := 0
	for _, r := range results {
		fmt.Printf("--- %s ---\n", r.ID)
		fmt.Printf("  rows        : %d vs %d\n", r.RowsA, r.RowsB)
		if r.OK() {
			fmt.Printf("  full-table  : identical sha256 %s\n", r.HashA[:16])
			continue
		}
		bad++
		for _, p := range r.Problems {
			fmt.Printf("  *** %s\n", p)
		}
		for _, d := range r.FirstDiff {
			fmt.Println(d)
		}
	}

	if bad > 0 {
		return fmt.Errorf("%d dataset(s) differ", bad)
	}
	fmt.Println("\nIdentical — rows, per-column non-NULL counts, schema and full-table hash.")
	return nil
}

func buildDatabases(root string, all, selected []registry.Dataset) error {
	p := databases.DefaultPaths(root)

	// Sweep first: a dataset dropped from the registry leaves its .db.gz behind,
	// and the site assembly copies the staging directory wholesale.
	if err := databases.Clean(p, all); err != nil {
		return err
	}

	bin, err := databases.BuildParser(p)
	if err != nil {
		return err
	}
	for _, d := range selected {
		if err := databases.Build(p, bin, d); err != nil {
			return err
		}
	}
	return nil
}

// repoRoot walks up from the working directory to the directory holding
// datasets.json, so the command works from anywhere in the tree.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "datasets.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no datasets.json found in any parent of the working directory")
		}
		dir = parent
	}
}
