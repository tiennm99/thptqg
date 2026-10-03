// Package site turns the web app and the staged databases into the directory
// GitHub Pages publishes.
//
// SvelteKit prerenders one HTML file per route — the hub and one per dataset in
// the registry — so every URL is already a real static file when this runs. All
// that is left is the 404 document and the checks on what shipped.
//
// GitHub Pages serves the prerendered files as directory indexes, which is why
// this needs no SPA fallback redirect. That matters beyond tidiness: the usual
// fallback rewrites the URL and would interfere with the ?q= deep links the app
// relies on.
package site

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tiennm99/thptqg/assembler/internal/databases"
	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

// Paths locates the pieces this package needs.
type Paths struct {
	// Web is the SvelteKit project directory.
	Web string
	// Dist is where the static adapter emits, inside the web workspace.
	Dist string
	// Site is the artifact the deploy action uploads, at the repository root.
	Site string
}

// DefaultPaths derives the standard layout from the repository root.
func DefaultPaths(root string) Paths {
	web := filepath.Join(root, "web")
	return Paths{
		Web:  web,
		Dist: filepath.Join(web, "dist"),
		Site: filepath.Join(root, "_site"),
	}
}

// BuildWeb runs the web build.
//
// Shelling out to npm is not a wart: SvelteKit is a Node tool, and web/ is the
// only npm project in the repository. This stage owns the sequencing, not the
// bundling.
func BuildWeb(p Paths) error {
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = p.Web
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("web build: %w", err)
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

	// The prerendered pages, _app/ and the gzipped databases in one move.
	if err := copyTree(p.Dist, p.Site); err != nil {
		return err
	}

	// Unknown paths render the hub rather than the default Pages 404. Asset URLs
	// in that file are absolute, so it works at any depth.
	if err := copyFile(index, filepath.Join(p.Site, "404.html")); err != nil {
		return err
	}

	if err := checkDatasetPages(p.Site, datasets); err != nil {
		return err
	}
	if err := checkDatabasesPresent(p.Site, datasets); err != nil {
		return err
	}
	if err := checkNoStrayArtifacts(p.Site); err != nil {
		return err
	}

	fmt.Printf("assembled %s\n", p.Site)
	fmt.Printf("  /thptqg/\n  /thptqg/404.html\n")
	for _, d := range datasets {
		fmt.Printf("  /thptqg/%s\n", d.ID)
	}
	return nil
}

// checkDatasetPages: every dataset must have prerendered an entry point.
//
// The pages come from the web build's entry generator reading the same
// registry, so a missing one means the two fell out of step — a dataset URL
// that 404s while the build stays green.
func checkDatasetPages(siteDir string, datasets []registry.Dataset) error {
	var missing []string
	for _, d := range datasets {
		if _, err := os.Stat(filepath.Join(siteDir, d.ID, "index.html")); err != nil {
			missing = append(missing, d.ID+"/index.html")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"the web build prerendered no page for: %s\n"+
				"Those URLs would 404. Check the entry generator in web/src/routes/[dataset]/+page.ts",
			strings.Join(missing, ", "))
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
		file := filepath.Join(siteDir, "db", d.ID+databases.Extension)
		st, err := os.Stat(file)
		if err != nil || st.Size() == 0 {
			missing = append(missing, d.ID+databases.Extension)
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

// strayArtifact matches what must never reach the output: a SQLite journal
// from an interrupted run, a database under the old .db name, or one under the
// .sqlite30 name a former range-request client asked for. Each is 100+ MB.
var strayArtifact = regexp.MustCompile(`(\.db|\.sqlite30?)(-journal|-wal|-shm)$|\.db$|\.sqlite30$|\.gz$`)

// checkNoStrayArtifacts rejects leftovers that would be published.
//
// The staging directory is copied wholesale, so anything an interrupted run left
// behind goes straight through — and each of these is 100+ MB, downloaded in
// full by anyone the site hands the wrong name to.
func checkNoStrayArtifacts(siteDir string) error {
	var stray []string
	err := filepath.WalkDir(siteDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strayArtifact.MatchString(d.Name()) {
			stray = append(stray, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(stray) > 0 {
		var b strings.Builder
		b.WriteString("stray database artefact(s) found in the site output:\n")
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
