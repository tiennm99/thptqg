// Package sources says where each dataset's spreadsheets are published and
// what to call them locally.
//
// It carries no link lists. Each source names the article that published the
// files; internal/article reads the links out of that page at run time, and
// internal/fetch moves the bytes.
package sources

import (
	"fmt"

	"github.com/tiennm99/thptqg/crawler/internal/article"
)

// Source is one crawlable dataset.
type Source struct {
	// ID is the dataset id: the subcommand, the directory under data/, and the
	// parser config name, all at once. Keeping it single means a source
	// cannot be pointed at the wrong dataset's directory.
	ID      string
	Summary string

	// Article is the published page listing this dataset's files.
	Article string

	// Exts are the file extensions to pick out of that page.
	Exts []string

	// Headers are sent with every request for this source, for the article and
	// the files alike.
	Headers map[string]string

	// WantFiles is how many links the article is expected to yield. A page that
	// suddenly yields fewer has changed shape, and silently crawling a partial
	// dataset is the failure this exists to prevent — parser would happily
	// build a short database and only the row-count guard would catch it, after
	// the fact.
	WantFiles int

	// Dest names the local file for one discovered link.
	//
	// This is the load-bearing part. parser sorts input files bytewise and
	// inserts last-wins, so the names chosen here decide which row survives a
	// duplicate exam number. Two sources answer it differently and both have a
	// reason: see source_2016.go and source_2017.go.
	Dest func(article.Link) (string, error)
}

// File is one spreadsheet to download.
type File struct {
	// Name is the human label used in progress output.
	Name string
	URL  string
	// Dest is the filename within the dataset directory.
	Dest string
}

// Resolve turns the links found on a source's article into the files to fetch,
// enforcing the count and rejecting any two links that would write to the same
// name — which would silently cost the dataset a file.
func (s Source) Resolve(links []article.Link) ([]File, error) {
	if len(links) != s.WantFiles {
		return nil, fmt.Errorf(
			"%s: found %d links on %s, expected %d — the page has changed shape; "+
				"check it before crawling, a partial dataset builds without complaint",
			s.ID, len(links), s.Article, s.WantFiles)
	}

	out := make([]File, 0, len(links))
	seen := make(map[string]string, len(links))
	for _, l := range links {
		dest, err := s.Dest(l)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.ID, err)
		}
		if prev, dup := seen[dest]; dup {
			return nil, fmt.Errorf("%s: %q and %q both name the local file %q",
				s.ID, prev, l.URL, dest)
		}
		seen[dest] = l.URL

		name := l.Text
		if name == "" {
			name = dest
		}
		out = append(out, File{Name: name, URL: l.URL, Dest: dest})
	}
	return out, nil
}

// registry is ordered: it drives the help output.
var registry = []Source{source2016, source2017}

// All returns every known source, in help order.
func All() []Source { return registry }

// Lookup finds a source by ID.
func Lookup(id string) (Source, error) {
	for _, s := range registry {
		if s.ID == id {
			return s, nil
		}
	}
	return Source{}, fmt.Errorf("unknown dataset %q", id)
}
