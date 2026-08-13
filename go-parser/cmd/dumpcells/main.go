// Command dumpcells emits the canonical cell rendering of a spreadsheet, for
// comparison against the Rust/calamine ground truth produced by
// parser/examples/dump_cells.rs.
//
// The canonical stream carries geometry and rendered cell values only. The
// calamine Data variant is deliberately excluded: Data::Empty and
// Data::String("") both render "" and both count as blank everywhere
// downstream, so the distinction cannot affect the database.
//
// Usage: dumpcells <spreadsheet> [out-file]
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tiennm99/thptqg/go-parser/internal/reader"
)

// escape mirrors the Rust dumper so field separators can never break the format.
func escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, ch := range s {
		switch ch {
		case '\\':
			b.WriteString(`\\`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dumpcells <spreadsheet> [out-file]")
		os.Exit(2)
	}
	path := os.Args[1]

	out := os.Stdout
	if len(os.Args) > 2 {
		f, err := os.Create(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "create: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}
	w := bufio.NewWriterSize(out, 1<<20)
	defer w.Flush()

	wb, err := reader.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer wb.Close()

	sheets := wb.Sheets()
	fmt.Fprintf(w, "FILE\t%s\n", escape(path))
	fmt.Fprintf(w, "SHEETCOUNT\t%d\n", len(sheets))

	for _, sh := range sheets {
		fmt.Fprintf(w, "SHEET\t%d\t%s\t%d\t%d\n", sh.Index, escape(sh.Name), sh.Height, sh.Width)
		err := wb.EachRow(sh.Index, func(s reader.Sheet, rowIdx int, row []reader.Cell) error {
			fmt.Fprintf(w, "ROW\t%d\t%d\t%d\n", s.Index, rowIdx, len(row))
			for c, cell := range row {
				fmt.Fprintf(w, "CELL\t%d\t%d\t%d\t%s\n", s.Index, rowIdx, c, escape(cell.Str))
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "rows: %v\n", err)
			os.Exit(1)
		}
	}
}
