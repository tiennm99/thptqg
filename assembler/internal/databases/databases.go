// Package databases builds, verifies and compresses one SQLite file per
// dataset.
//
// VERIFICATION IS THE POINT OF THIS PACKAGE, not an extra.
//
// Nothing between the parser and the published site otherwise asserts that a
// database has data in it. The parser logs a file-level failure and continues,
// returns success regardless, and finishes cleanly even at zero rows; the site
// assembly only inspects filenames. So a reader that silently under-produced
// would publish a truncated dataset with green CI and no red signal anywhere.
//
// The guards below close that: a build whose row count does not match the
// registry, or whose artifact is implausibly small, fails the pipeline.
package databases

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go driver: the pipeline stays cgo-free

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

// driverName is modernc.org/sqlite's registered name.
const driverName = "sqlite"

// minSizeRatio: a gzipped database far below its usual size means a truncated
// build, even if the row count somehow passed.
const minSizeRatio = 0.9

// Paths locates the pieces this package needs.
type Paths struct {
	// Root is the repository root.
	Root string
	// Parser is the parser module directory.
	Parser string
	// OutDir is where the databases are staged — the directory Vite publishes.
	OutDir string
}

// DefaultPaths derives the standard layout from the repository root.
func DefaultPaths(root string) Paths {
	return Paths{
		Root:   root,
		Parser: filepath.Join(root, "parser"),
		OutDir: filepath.Join(root, ".build", "public", "db"),
	}
}

// BuildParser compiles the parser binary and returns its path.
//
// Compiling here rather than expecting a prebuilt binary keeps the pipeline one
// command. Go caches the work, so repeat runs cost almost nothing.
func BuildParser(p Paths) (string, error) {
	bin := filepath.Join(p.Parser, "bin", "xlsxread")
	cmd := exec.Command("go", "-C", p.Parser, "build", "-o", "bin/xlsxread", "./cmd/xlsxread")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("compiling the parser: %w", err)
	}
	return bin, nil
}

// Build runs the parser for one dataset, verifies the result and compresses it.
//
// Only the .gz survives: shipping a 100+ MB uncompressed database is made
// structurally impossible rather than left to a cleanup step.
func Build(p Paths, bin string, d registry.Dataset) error {
	if err := os.MkdirAll(p.OutDir, 0o755); err != nil {
		return err
	}
	db := filepath.Join(p.OutDir, d.ID+".db")

	cmd := exec.Command(bin,
		"build",
		"--schema", filepath.Join(p.Parser, "configs", d.ID+".yml"),
		"--input", filepath.Join(p.Root, "data", d.ID),
		"--output", db,
	)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: parser failed: %w", d.ID, err)
	}

	rows, err := countRows(db)
	if err != nil {
		return fmt.Errorf("%s: %w", d.ID, err)
	}
	if rows != d.ExpectedRows {
		return fmt.Errorf(
			"%s: row count %d, expected %d\nRefusing to publish — the build did not reproduce the known dataset",
			d.ID, rows, d.ExpectedRows)
	}
	fmt.Printf("  ✓ %s: %d rows (matches expected)\n", d.ID, rows)

	gz, size, err := compress(db)
	if err != nil {
		return fmt.Errorf("%s: %w", d.ID, err)
	}

	sizeMb := float64(size) / 1024 / 1024
	if min := d.DbSizeMb * minSizeRatio; sizeMb < min {
		return fmt.Errorf(
			"%s: %.1f MB is below %.1f MB (%.0f%% of the expected %.0f MB)\n"+
				"Refusing to publish — the artifact looks truncated",
			d.ID, sizeMb, min, minSizeRatio*100, d.DbSizeMb)
	}

	fmt.Printf("  → %s (%.1f MB)\n\n", filepath.Base(gz), sizeMb)
	return nil
}

// countRows opens the database read-only and counts what was written.
func countRows(path string) (int64, error) {
	conn, err := sql.Open(driverName, "file:"+path+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	var n int64
	if err := conn.QueryRow("SELECT COUNT(*) FROM student").Scan(&n); err != nil {
		return 0, fmt.Errorf("counting rows: %w", err)
	}
	return n, nil
}

// compress gzips path to path+".gz" and removes the original, returning the
// compressed path and its size.
//
// The source is deleted only after the compressed file is closed successfully,
// so a failure part-way through leaves the database rather than losing it.
func compress(path string) (string, int64, error) {
	in, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()

	gzPath := path + ".gz"
	out, err := os.Create(gzPath)
	if err != nil {
		return "", 0, err
	}

	zw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		out.Close()
		return "", 0, err
	}
	if _, err := io.Copy(zw, in); err != nil {
		zw.Close()
		out.Close()
		os.Remove(gzPath)
		return "", 0, err
	}
	if err := zw.Close(); err != nil {
		out.Close()
		os.Remove(gzPath)
		return "", 0, err
	}
	if err := out.Close(); err != nil {
		os.Remove(gzPath)
		return "", 0, err
	}

	if err := in.Close(); err != nil {
		return "", 0, err
	}
	if err := os.Remove(path); err != nil {
		return "", 0, fmt.Errorf("removing the uncompressed database: %w", err)
	}

	st, err := os.Stat(gzPath)
	if err != nil {
		return "", 0, err
	}
	return gzPath, st.Size(), nil
}

// Clean removes staged artifacts for datasets that are no longer in the
// registry. Without this a removed dataset's .db.gz lingers in the staging
// directory, and the site assembly copies that directory wholesale — so the
// dead database would be published again.
func Clean(p Paths, keep []registry.Dataset) error {
	entries, err := os.ReadDir(p.OutDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	wanted := make(map[string]bool, len(keep)*2)
	for _, d := range keep {
		wanted[d.ID+".db"] = true
		wanted[d.ID+".db.gz"] = true
	}

	for _, e := range entries {
		if e.IsDir() || wanted[e.Name()] {
			continue
		}
		full := filepath.Join(p.OutDir, e.Name())
		if err := os.Remove(full); err != nil {
			return err
		}
		fmt.Printf("  removed stale artifact %s\n", e.Name())
	}
	return nil
}
