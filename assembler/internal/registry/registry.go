// Package registry reads the repository-root datasets.json.
//
// That file is the one place every stage agrees on what exists. The Vite app
// reads it too, which is why it is JSON: Go and the browser both parse it
// without a dependency.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Dataset is one entry in the registry.
type Dataset struct {
	ID string `json:"id"`
	// ExpectedRows is exact. The inputs are frozen historical exam results, so
	// a deviation of even one row means something changed that nobody intended.
	ExpectedRows int64 `json:"expectedRows"`
	// DbSizeMb is the usual size of the gzipped database, used to catch a
	// build that produced a plausible row count but a truncated artifact.
	DbSizeMb float64 `json:"dbSizeMb"`
}

type file struct {
	Datasets []Dataset `json:"datasets"`
}

// Load reads datasets.json from the repository root.
func Load(root string) ([]Dataset, error) {
	path := filepath.Join(root, "datasets.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the dataset registry: %w", err)
	}

	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(f.Datasets) == 0 {
		return nil, fmt.Errorf("%s declares no datasets", path)
	}

	for _, d := range f.Datasets {
		switch {
		case d.ID == "":
			return nil, fmt.Errorf("%s: a dataset has no id", path)
		case d.ExpectedRows <= 0:
			return nil, fmt.Errorf("%s: %s has no expectedRows; the build guard needs it", path, d.ID)
		case d.DbSizeMb <= 0:
			return nil, fmt.Errorf("%s: %s has no dbSizeMb; the size guard needs it", path, d.ID)
		}
	}
	return f.Datasets, nil
}

// Select returns the named datasets, or all of them when none are named.
func Select(all []Dataset, ids []string) ([]Dataset, error) {
	if len(ids) == 0 {
		return all, nil
	}
	byID := make(map[string]Dataset, len(all))
	known := make([]string, 0, len(all))
	for _, d := range all {
		byID[d.ID] = d
		known = append(known, d.ID)
	}

	out := make([]Dataset, 0, len(ids))
	for _, id := range ids {
		d, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("unknown dataset %q (known: %v)", id, known)
		}
		out = append(out, d)
	}
	return out, nil
}
