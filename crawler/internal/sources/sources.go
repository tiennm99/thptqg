// Package sources describes where each dataset's spreadsheets come from.
//
// A source is a dataset id plus the list of remote files that fill it.
// Downloading is internal/fetch's job; deciding what to download and what to
// call it locally is this package's.
package sources

import "fmt"

// File is one remote spreadsheet.
type File struct {
	// Name is the human label used in progress output, e.g. "An Giang".
	Name string
	URL  string
	// Dest is the filename within the dataset directory.
	//
	// This is not cosmetic. go-parser sorts its input files and inserts with
	// INSERT OR REPLACE, which is last-wins, so filenames decide which row
	// survives a duplicate exam number. A re-crawl that names files differently
	// can produce a database with the same row count and different content.
	Dest string
}

// Source is one crawlable dataset.
type Source struct {
	// ID is the dataset id: the subcommand, the directory under data/, and the
	// go-parser config name, all at once. Keeping it single means a source
	// cannot be pointed at the wrong dataset's directory.
	ID      string
	Summary string
	// Headers are sent with every request for this source.
	Headers map[string]string
	// Files lists what to download. It returns an error rather than an empty
	// list when a source is known but not yet configured, so "nothing to do"
	// can never be mistaken for success.
	Files func() ([]File, error)
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
