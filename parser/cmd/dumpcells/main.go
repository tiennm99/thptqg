// Command dumpcells emits the canonical cell rendering of a spreadsheet.
//
// It exists to diagnose a TestReaderFidelity failure. That suite compares a
// SHA-256 per input file against the frozen oracle, so a mismatch names the
// file but not the cell; this prints the stream the hash is taken over — plus a
// leading FILE line, which the hash excludes — so a dump from before and after
// a reader change can be diffed line by line.
//
// The canonical stream carries geometry and rendered cell values only. Whether
// a cell was absent or an empty string is deliberately excluded: both render ""
// and both count as blank everywhere downstream, so the distinction cannot
// affect the database.
//
// Usage: dumpcells <spreadsheet> [out-file]
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tiennm99/thptqg/parser/internal/reader"
)

// escape hides the field and record separators so a cell value can never break
// the format. Must stay in step with the fidelity test's escaping.
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
