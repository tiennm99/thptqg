package sources

import (
	"fmt"

	"github.com/tiennm99/thptqg/crawler/internal/article"
)

// source2016 fetches the 2016 dataset: one spreadsheet per exam cluster
// (cụm thi), 4 .xls and 115 .xlsx.
//
// The article is a school site that aggregated every cluster's file. It is
// still online, and its links are read at crawl time like any other source.
//
// Dest keeps the server's own filename verbatim — a 32-hex content hash, the
// cluster slug, then a millisecond timestamp. Two reasons not to prettify it:
//
//   - parser sorts inputs bytewise and inserts last-wins, so filenames decide
//     which row survives a duplicate exam number. That is live here, not
//     hypothetical: 877,464 source rows collapse to 877,461, so three rows'
//     contents depend on this ordering.
//   - parser/testdata/reader-fidelity-hashes.tsv is keyed by full path, and
//     it is frozen — it was produced by the Rust reader, which no longer exists.
//
// Unlike 2017 this needs no transliteration: the name comes from the URL, so it
// is exact by construction rather than derived from a label.
var source2016 = Source{
	ID:        "2016",
	Summary:   "119 exam-cluster files (4 .xls + 115 .xlsx)",
	Article:   "https://dtnt.bacninh.edu.vn/tin-tuc/tin-tuc-su-kien/cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html",
	Exts:      []string{".xls", ".xlsx"},
	WantFiles: 119,
	Dest: func(l article.Link) (string, error) {
		if l.File == "" {
			return "", fmt.Errorf("link %s has no filename", l.URL)
		}
		return l.File, nil
	},
}
