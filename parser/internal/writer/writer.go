// Package writer handles SQLite output: DDL setup, INSERT OR REPLACE, VACUUM
// and the stats block — a port of parser/src/writer.rs.
//
// Every dataset writes the same canonical table (internal/schema), so there is
// exactly one insert path. Columns a dataset carries no data for bind NULL.
//
// The stats lines are reproduced verbatim because they are the operator-facing
// output of the build, and docs/deployment-guide.md points at the per-file row
// counts for troubleshooting.
package writer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tiennm99/thptqg/parser/internal/schema"
	"github.com/tiennm99/thptqg/parser/internal/sqlitedb"
	"github.com/tiennm99/thptqg/parser/internal/transform"
)

// OpenDB deletes any existing database at dbPath, recreates it, and executes the
// canonical DDL.
//
// Deleting the file rather than issuing DROP TABLE mirrors build-lib.js:54 via
// writer.rs:24-30. A consequence worth knowing: a concurrent reader sees the file
// vanish mid-rebuild rather than a transactional swap.
func OpenDB(dbPath string) (*sql.DB, error) {
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return nil, fmt.Errorf("remove existing db %s: %w", dbPath, err)
		}
	}
	if parent := filepath.Dir(dbPath); parent != "" && parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", parent, err)
		}
	}

	db, err := sql.Open(sqlitedb.DriverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db %s: %w", dbPath, err)
	}
	if _, err := db.Exec(schema.DDL); err != nil {
		db.Close()
		return nil, fmt.Errorf("execute DDL: %w", err)
	}
	return db, nil
}

// Inserter wraps a prepared INSERT statement.
//
// Rust calls conn.execute(INSERT_SQL, ...) per row (writer.rs:73), re-preparing
// each time. Preparing once is a performance choice, not a parity requirement —
// the SQL and its bindings are identical either way.
type Inserter struct{ stmt *sql.Stmt }

// Prepare compiles the canonical INSERT against tx.
func Prepare(tx *sql.Tx) (*Inserter, error) {
	stmt, err := tx.Prepare(schema.InsertSQL)
	if err != nil {
		return nil, fmt.Errorf("prepare insert: %w", err)
	}
	return &Inserter{stmt: stmt}, nil
}

// Close releases the prepared statement.
func (i *Inserter) Close() error { return i.stmt.Close() }

// Insert binds one parsed row and executes the INSERT.
//
// Parameter order is IdentityFields then ScoreFields. Subjects absent from
// row.Scores — and the two identity columns only the 2016 layouts populate —
// bind NULL.
func (i *Inserter) Insert(row *transform.ParsedRow) error {
	args := make([]any, 0, schema.ParamCount)
	args = append(args,
		row.SoBaoDanh,
		row.HoTen,
		row.HoTenAscii,
		nullableString(row.NgaySinh),
		nullableString(row.TenCumThi),
		nullableString(row.GioiTinh),
	)
	for _, field := range schema.ScoreFields {
		if v, ok := row.Scores[field]; ok {
			args = append(args, v)
		} else {
			args = append(args, nil)
		}
	}
	if len(args) != schema.ParamCount {
		return fmt.Errorf("built %d params, want %d", len(args), schema.ParamCount)
	}
	if _, err := i.stmt.Exec(args...); err != nil {
		return err
	}
	return nil
}

func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// Stats carries the counters the build loop accumulates.
type Stats struct {
	SourceRows uint64
	Skipped    uint64
	Errors     uint64
}

// Finish runs VACUUM and prints the stats block.
//
// VACUUM must run AFTER the transaction commits — SQLite refuses it inside one.
//
// The wording branches on datasetLabel, which Rust derives from the input
// directory's basename (main.rs:98-101). That makes the output depend on a
// filesystem path rather than on config; it is reproduced here for parity, and
// the caller passes the label explicitly so tests are not at the mercy of a
// temp-directory name.
func Finish(db *sql.DB, dbPath string, st Stats, datasetLabel string, isOld2 bool) error {
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}

	var dbCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM student").Scan(&dbCount); err != nil {
		return fmt.Errorf("count rows: %w", err)
	}

	insertable := st.SourceRows - st.Skipped

	fmt.Println()
	if isOld2 {
		fmt.Printf("Source non-blank data rows:      %d\n", st.SourceRows)
		fmt.Printf("  skipped (empty/non-numeric SBD): %d\n", st.Skipped)
	} else {
		fmt.Printf("Source data rows (post-header):  %d\n", st.SourceRows)
		if containsOld(datasetLabel) {
			fmt.Printf("  skipped (empty/non-numeric SBD): %d\n", st.Skipped)
		} else {
			fmt.Printf("  skipped (empty/invalid):        %d\n", st.Skipped)
		}
	}
	fmt.Printf("  insertable:                     %d\n", insertable)
	fmt.Printf("  insert errors:                  %d\n", st.Errors)
	fmt.Printf("DB rows (distinct SBD):           %d\n", dbCount)

	if !containsOld(datasetLabel) && st.Errors == 0 {
		gap := int64(insertable) - dbCount
		if gap == 0 {
			fmt.Println("Audit: OK — every source row made it in.")
		} else {
			fmt.Printf("Audit: %d row(s) collapsed (duplicate SBDs overwriting).\n", gap)
		}
	}

	var size int64
	if fi, err := os.Stat(dbPath); err == nil {
		size = fi.Size()
	}
	fmt.Printf("Size: %.1f MB\n", float64(size)/1024.0/1024.0)
	return nil
}

func containsOld(label string) bool {
	for i := 0; i+3 <= len(label); i++ {
		if label[i:i+3] == "old" {
			return true
		}
	}
	return false
}
