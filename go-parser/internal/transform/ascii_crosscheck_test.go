package transform_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/tiennm99/thptqg/go-parser/internal/sqlitedb"
	"github.com/tiennm99/thptqg/go-parser/internal/transform"
)

// TestToAsciiAgainstRustOutput cross-checks ToAscii against Rust on real data.
//
// A Rust-built database is its own oracle: every row carries ho_ten alongside
// the ho_ten_ascii that Rust derived from it, so the whole table is a
// name -> expected-slug corpus far broader than the 20 hand-picked unit cases.
//
// Point it at a Rust-built database:
//
//	GO_PARSER_RUST_DB=/tmp/rust-2016.db go test ./internal/transform/
//
// Skips when unset, so the default suite stays hermetic.
func TestToAsciiAgainstRustOutput(t *testing.T) {
	path := os.Getenv("GO_PARSER_RUST_DB")
	if path == "" {
		t.Skip("GO_PARSER_RUST_DB not set; skipping cross-check against Rust output")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("GO_PARSER_RUST_DB=%s not readable: %v", path, err)
	}

	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT ho_ten, ho_ten_ascii FROM student")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var checked, bad int
	for rows.Next() {
		var name, rustAscii string
		if err := rows.Scan(&name, &rustAscii); err != nil {
			t.Fatalf("scan: %v", err)
		}
		checked++
		if got := transform.ToAscii(name); got != rustAscii {
			bad++
			if bad <= 5 {
				t.Errorf("ToAscii(%q)\n  rust = %q\n  go   = %q", name, rustAscii, got)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate: %v", err)
	}
	if checked == 0 {
		t.Fatal("database contained no rows")
	}
	t.Logf("compared %d real names, %d mismatches", checked, bad)
}
