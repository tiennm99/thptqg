// Command assemble turns source data and the web app into the directory
// GitHub Pages publishes.
//
//	assemble            # databases, then the site
//	assemble db         # databases only (add ids to limit: assemble db 2017)
//	assemble site       # web build and _site only, reusing staged databases
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
  (none)   databases, then the site
  db       build, verify and compress the databases
  site     build the web app and assemble _site

Naming datasets limits the db step to those; the site step always covers all of
them, since a partial site would publish links to databases it did not build.
`)
}

func run(args []string) error {
	step := ""
	if len(args) > 0 {
		switch args[0] {
		case "db", "site":
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
