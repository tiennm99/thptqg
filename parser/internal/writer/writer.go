// Package writer handles SQLite output: DDL setup, INSERT OR REPLACE, VACUUM
// and the stats block.
//
// Every dataset writes the same canonical table (internal/schema), so there is
// exactly one insert path. Columns a dataset carries no data for bind NULL.
//
// The stats lines are the operator-facing output of the build, and
// docs/deployment-guide.md points at the per-file row counts for
// troubleshooting — do not reword them casually.
package writer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tiennm99/thptqg/parser/internal/schema"
	"github.com/tiennm99/thptqg/parser/internal/sqlitedb"
	"github.com/tiennm99/thptqg/parser/internal/transform"
)

// OpenDB deletes any existing database at dbPath, recreates it, and executes the
// canonical DDL.
//
// Deleting the file rather than issuing DROP TABLE has a consequence worth
// knowing: a concurrent reader sees the file vanish mid-rebuild rather than a
// transactional swap.
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

// Inserter wraps the canonical INSERT, prepared once per transaction rather
// than per row.
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

// BuildNameIndex fills name_word from the student rows, one entry per distinct
// word of each ASCII name.
//
// A second pass rather than a write alongside each insert: a repeated exam
// number replaces its earlier row, and the words of the row it replaced would
// otherwise stay behind pointing at a name that is no longer there.
func BuildNameIndex(db *sql.DB) error {
	rows, err := db.Query("SELECT so_bao_danh, ho_ten_ascii FROM student")
	if err != nil {
		return fmt.Errorf("read names: %w", err)
	}
	defer rows.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin name index: %w", err)
	}
	stmt, err := tx.Prepare(schema.NameWordInsertSQL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare name index: %w", err)
	}

	var words uint64
	seen := make(map[string]struct{}, 8)
	for rows.Next() {
		var sbd, ascii string
		if err := rows.Scan(&sbd, &ascii); err != nil {
			tx.Rollback()
			return fmt.Errorf("scan name: %w", err)
		}
		clear(seen)
		for _, w := range strings.Fields(ascii) {
			if _, dup := seen[w]; dup {
				continue
			}
			seen[w] = struct{}{}
			if _, err := stmt.Exec(w, sbd, ascii); err != nil {
				tx.Rollback()
				return fmt.Errorf("insert name word: %w", err)
			}
			words++
		}
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return fmt.Errorf("read names: %w", err)
	}
	if err := stmt.Close(); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit name index: %w", err)
	}

	if _, err := db.Exec(schema.PostLoadSQL); err != nil {
		return fmt.Errorf("post-load statements: %w", err)
	}
	fmt.Printf("Name index:                       %d words\n", words)
	return nil
}

// Finish builds the derived tables, runs VACUUM and prints the stats block.
//
// VACUUM must run AFTER the transaction commits — SQLite refuses it inside one
// — and after the name index, so the file is laid out in one pass.
func Finish(db *sql.DB, dbPath string, st Stats) error {
	if err := BuildNameIndex(db); err != nil {
		return err
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}

	var dbCount int64
	if err := db.QueryRow("SELECT COUNT(*) FROM student").Scan(&dbCount); err != nil {
		return fmt.Errorf("count rows: %w", err)
	}

	insertable := st.SourceRows - st.Skipped

	fmt.Println()
	fmt.Printf("Source data rows (post-header):  %d\n", st.SourceRows)
	fmt.Printf("  skipped (empty/invalid):        %d\n", st.Skipped)
	fmt.Printf("  insertable:                     %d\n", insertable)
	fmt.Printf("  insert errors:                  %d\n", st.Errors)
	fmt.Printf("DB rows (distinct SBD):           %d\n", dbCount)

	if st.Errors == 0 {
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
