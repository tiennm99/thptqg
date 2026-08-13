// Package site turns the web app and the staged databases into the directory
// GitHub Pages publishes.
//
// The app resolves its dataset from the URL, so every page is the same
// index.html. Because Vite's `base` is absolute (/thptqg/), that file references
// /thptqg/assets/... no matter which directory it is served from — so copying it
// to each dataset path produces a real static file at every URL.
//
// GitHub Pages serves those as directory indexes, which is why this needs no
// SPA 404-fallback redirect. That matters beyond tidiness: the usual fallback
// rewrites the URL and would interfere with the ?q= deep links the app relies
// on.
package site

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

// Paths locates the pieces this package needs.
type Paths struct {
	Root string
	// Web is the Vite project directory.
	Web string
	// Dist is where Vite emits, inside the web workspace.
	Dist string
	// Site is the artifact the deploy action uploads, at the repository root.
	Site string
}

// DefaultPaths derives the standard layout from the repository root.
func DefaultPaths(root string) Paths {
	web := filepath.Join(root, "web")
	return Paths{
		Root: root,
		Web:  web,
		Dist: filepath.Join(web, "dist"),
		Site: filepath.Join(root, "_site"),
	}
}

// BuildWeb runs the Vite build.
//
// Shelling out to npm is not a wart: Vite is a Node tool, and web/ is the only
// npm project left in the repository. This stage owns the sequencing, not the
// bundling.
func BuildWeb(p Paths) error {
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = p.Web
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vite build: %w", err)
	}
	return nil
}

// Assemble copies the build to one directory per dataset and checks the result.
func Assemble(p Paths, datasets []registry.Dataset) error {
	index := filepath.Join(p.Dist, "index.html")
	if _, err := os.Stat(index); err != nil {
		return fmt.Errorf("no build found at %s — run the web build first", p.Dist)
	}

	if err := os.RemoveAll(p.Site); err != nil {
		return err
	}
	if err := os.MkdirAll(p.Site, 0o755); err != nil {
		return err
	}

	// Base build: index.html, assets/, and the gzipped databases from publicDir.
	if err := copyTree(p.Dist, p.Site); err != nil {
		return err
	}

	// Unknown paths render the hub rather than the default Pages 404.
	if err := copyFile(index, filepath.Join(p.Site, "404.html")); err != nil {
		return err
	}

	// One entry point per dataset.
	for _, d := range datasets {
		dir := filepath.Join(p.Site, d.ID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := copyFile(index, filepath.Join(dir, "index.html")); err != nil {
			return err
		}
	}

	if err := checkDatabasesPresent(p.Site, datasets); err != nil {
		return err
	}
	if err := checkNoRawDatabases(p.Site); err != nil {
		return err
	}

	fmt.Printf("assembled %s\n", p.Site)
	fmt.Printf("  /thptqg/\n  /thptqg/404.html\n")
	for _, d := range datasets {
		fmt.Printf("  /thptqg/%s\n", d.ID)
	}
	return nil
}

// checkDatabasesPresent: every dataset must have shipped its database.
//
// Without this the site assembles happily with an empty db/ directory — every
// page renders, every query 404s, and CI stays green. That is the failure this
// catches; the size and row-count guards only run when a database was built at
// all.
func checkDatabasesPresent(siteDir string, datasets []registry.Dataset) error {
	var missing []string
	for _, d := range datasets {
		gz := filepath.Join(siteDir, "db", d.ID+".db.gz")
		st, err := os.Stat(gz)
		if err != nil || st.Size() == 0 {
			missing = append(missing, d.ID+".db.gz")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"no database in the site output for: %s\n"+
				"Every page would render and every query would 404. Build the databases first",
			strings.Join(missing, ", "))
	}
	return nil
}

// rawDatabase matches an uncompressed SQLite artifact, including the temporary
// files SQLite leaves mid-build.
var rawDatabase = regexp.MustCompile(`\.db(-journal|-wal|-shm)?$`)

// checkNoRawDatabases rejects an uncompressed database that reached the output.
//
// The build gzips without keeping the source, so none should exist — but the
// staging directory is copied wholesale, and a leftover from an interrupted run
// would go straight through. A raw database is 100+ MB.
func checkNoRawDatabases(siteDir string) error {
	var stray []string
	err := filepath.WalkDir(siteDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && rawDatabase.MatchString(d.Name()) {
			stray = append(stray, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(stray) > 0 {
		var b strings.Builder
		b.WriteString("uncompressed database artefact(s) found in the site output:\n")
		for _, f := range stray {
			st, _ := os.Stat(f)
			fmt.Fprintf(&b, "  %s (%.1f MB)\n", f, float64(st.Size())/1048576)
		}
		b.WriteString("remove them from the staging directory and re-run")
		return fmt.Errorf("%s", b.String())
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
