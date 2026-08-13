// Command xlsxread reads .xls/.xlsx files and builds SQLite databases for the
// thptqg datasets.
//
// The CLI contract is fixed by parser/scripts/build-db.js and must match the
// Rust binary exactly:
//
//	xlsxread build --schema <config.yml> --input <dir> --output <db>
//	xlsxread audit --schema <config.yml> --input <dir> --db <db>
//
// Implemented with the standard flag package rather than a CLI framework: two
// subcommands with three flags each do not justify a dependency.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tiennm99/thptqg/parser/internal/audit"
	"github.com/tiennm99/thptqg/parser/internal/config"
	"github.com/tiennm99/thptqg/parser/internal/ingest"
)

func usage() {
	fmt.Fprint(os.Stderr, `xlsxread — read .xls/.xlsx files and build SQLite databases for thptqg datasets

Usage:
  xlsxread build --schema <config.yml> --input <dir> --output <db>
  xlsxread audit --schema <config.yml> --input <dir> --db <db>
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "build":
		runBuild(os.Args[2:])
	case "audit":
		runAudit(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	schemaPath := fs.String("schema", "", "path to the dataset YAML config file")
	inputDir := fs.String("input", "", "directory containing the .xls / .xlsx source files")
	outputPath := fs.String("output", "", "output SQLite database path")
	fs.Parse(args)

	if *schemaPath == "" || *inputDir == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "build requires --schema, --input and --output")
		os.Exit(2)
	}

	cfg, err := config.Load(*schemaPath)
	if err != nil {
		fatalf("Failed to load config: %v", err)
	}

	// The 2016 dataset selects its column layout per file at runtime; every other
	// dataset uses the fixed columns: mapping (main.rs:63-67).
	if cfg.FormatDetection != nil && *cfg.FormatDetection == "thptqg2016" {
		if err := ingest.Detect2016(cfg, *inputDir, *outputPath); err != nil {
			fatalf("%v", err)
		}
		return
	}
	if err := ingest.Standard(cfg, *inputDir, *outputPath); err != nil {
		fatalf("%v", err)
	}
}

func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	schemaPath := fs.String("schema", "", "path to the dataset YAML config file")
	inputDir := fs.String("input", "", "directory containing the .xlsx source files")
	dbPath := fs.String("db", "", "SQLite database to compare against")
	fs.Parse(args)

	if *schemaPath == "" || *inputDir == "" || *dbPath == "" {
		fmt.Fprintln(os.Stderr, "audit requires --schema, --input and --db")
		os.Exit(2)
	}

	cfg, err := config.Load(*schemaPath)
	if err != nil {
		fatalf("Failed to load config: %v", err)
	}
	res, err := audit.Run(*inputDir, *dbPath, cfg)
	if err != nil {
		fatalf("%v", err)
	}
	audit.PrintReport(res)

	// A mismatch is a non-zero exit even though the audit itself succeeded
	// (main.rs:46-48) — CI treats it as a failure signal.
	if !res.Matched {
		os.Exit(1)
	}
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
	os.Exit(1)
}
