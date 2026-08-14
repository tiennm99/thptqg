// Package verify compares two sets of built databases field by field.
//
// Nothing else checks database *content*. The reader-fidelity oracle covers
// reading the spreadsheets, and the row-count guard covers how many rows came
// out — but a change in transform or writer logic can alter what is in those
// rows while both of those still pass. This is what catches that.
//
// It works on any two builds: either side of a refactor, or a re-crawl against
// the databases already published.
package verify

import (
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/tiennm99/thptqg/assembler/internal/registry"
)

const driverName = "sqlite"

// Result is the outcome for one dataset.
type Result struct {
	ID        string
	Problems  []string
	RowsA     int64
	RowsB     int64
	HashA     string
	HashB     string
	FirstDiff []string
}

// OK reports whether the two databases are logically identical.
func (r Result) OK() bool { return len(r.Problems) == 0 }

// Compare checks every dataset in the registry, reading <dir>/<id>.db.gz (or
// <id>.db) from each side.
func Compare(datasets []registry.Dataset, dirA, dirB string) ([]Result, error) {
	out := make([]Result, 0, len(datasets))
	for _, d := range datasets {
		r, err := compareOne(d.ID, dirA, dirB)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func compareOne(id, dirA, dirB string) (Result, error) {
	res := Result{ID: id}

	a, cleanA, err := open(dirA, id)
	if err != nil {
		return res, err
	}
	defer cleanA()
	defer a.Close()

	b, cleanB, err := open(dirB, id)
	if err != nil {
		return res, err
	}
	defer cleanB()
	defer b.Close()

	sigA, err := schemaSignature(a)
	if err != nil {
		return res, err
	}
	sigB, err := schemaSignature(b)
	if err != nil {
		return res, err
	}
	if sigA != sigB {
		res.Problems = append(res.Problems, "schema/index metadata differs")
	}

	cols, err := columns(a)
	if err != nil {
		return res, err
	}

	if res.RowsA, err = count(a, "SELECT COUNT(*) FROM student"); err != nil {
		return res, err
	}
	if res.RowsB, err = count(b, "SELECT COUNT(*) FROM student"); err != nil {
		return res, err
	}
	if res.RowsA != res.RowsB {
		res.Problems = append(res.Problems, fmt.Sprintf("row count %d vs %d", res.RowsA, res.RowsB))
	}

	// Per-column non-NULL counts localise a difference to a column even when the
	// full-table hash has already said "something differs".
	for _, c := range cols {
		q := fmt.Sprintf("SELECT COUNT(%s) FROM student", c)
		na, err := count(a, q)
		if err != nil {
			return res, err
		}
		nb, err := count(b, q)
		if err != nil {
			return res, err
		}
		if na != nb {
			res.Problems = append(res.Problems, fmt.Sprintf("non-NULL count for %s: %d vs %d", c, na, nb))
		}
	}

	if res.HashA, err = tableHash(a, cols); err != nil {
		return res, err
	}
	if res.HashB, err = tableHash(b, cols); err != nil {
		return res, err
	}
	if res.HashA != res.HashB {
		res.Problems = append(res.Problems, "full-table hash differs")
		if res.FirstDiff, err = firstDifferences(a, b, cols, 20); err != nil {
			return res, err
		}
	}

	return res, nil
}

// open finds <id>.db.gz or <id>.db in dir and returns a handle. A compressed
// database is expanded to a temporary file, since SQLite needs to seek.
func open(dir, id string) (*sql.DB, func(), error) {
	noop := func() {}

	plain := filepath.Join(dir, id+".db")
	if _, err := os.Stat(plain); err == nil {
		db, err := sql.Open(driverName, "file:"+plain+"?mode=ro")
		return db, noop, err
	}

	gzPath := filepath.Join(dir, id+".db.gz")
	f, err := os.Open(gzPath)
	if err != nil {
		return nil, noop, fmt.Errorf("no %s.db or %s.db.gz in %s", id, id, dir)
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, noop, fmt.Errorf("%s: %w", gzPath, err)
	}
	defer zr.Close()

	tmp, err := os.CreateTemp("", "verify-"+id+"-*.db")
	if err != nil {
		return nil, noop, err
	}
	cleanup := func() { os.Remove(tmp.Name()) }

	if _, err := io.Copy(tmp, zr); err != nil {
		tmp.Close()
		cleanup()
		return nil, noop, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, noop, err
	}

	db, err := sql.Open(driverName, "file:"+tmp.Name()+"?mode=ro")
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	return db, cleanup, nil
}

func count(db *sql.DB, query string) (int64, error) {
	var n int64
	err := db.QueryRow(query).Scan(&n)
	return n, err
}

// columns returns the column names in declaration order.
func columns(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM pragma_table_info('student') ORDER BY cid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no student table")
	}
	return out, nil
}

// schemaSignature reduces the table and index metadata to a comparable string.
func schemaSignature(db *sql.DB) (string, error) {
	rows, err := db.Query("SELECT cid, name, type, \"notnull\", pk FROM pragma_table_info('student') ORDER BY cid")
	if err != nil {
		return "", err
	}
	var cols []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		if err := rows.Scan(&cid, &name, &typ, &notnull, &pk); err != nil {
			rows.Close()
			return "", err
		}
		cols = append(cols, fmt.Sprintf("%d:%s:%s:%d:%d", cid, name, typ, notnull, pk))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}

	idxRows, err := db.Query("SELECT name, \"unique\", partial FROM pragma_index_list('student')")
	if err != nil {
		return "", err
	}
	defer idxRows.Close()
	var idx []string
	for idxRows.Next() {
		var name string
		var uniq, partial int
		if err := idxRows.Scan(&name, &uniq, &partial); err != nil {
			return "", err
		}
		idx = append(idx, fmt.Sprintf("%s:%d:%d", name, uniq, partial))
	}
	if err := idxRows.Err(); err != nil {
		return "", err
	}
	// Index order is not guaranteed by SQLite; sort so it cannot cause a false
	// difference.
	sort.Strings(idx)

	return strings.Join(cols, "|") + "\n" + strings.Join(idx, "|"), nil
}

// Field and record separators. The values are hashed with explicit delimiters
// so that ("ab","c") and ("a","bc") cannot produce the same digest — joining
// them bare would make a shifted field boundary invisible.
const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// ser renders one value canonically.
//
// NULL gets a sentinel no real value can collide with, and a whole-numbered
// REAL is rendered with one decimal place so that a column holding 3 and one
// holding 3.0 compare equal, as SQLite considers them.
func ser(v any) string {
	switch t := v.(type) {
	case nil:
		return "\x00NULL"
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if t == math.Trunc(t) && !math.IsInf(t, 0) {
			return strconv.FormatFloat(t, 'f', 1, 64)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

// tableHash is a rolling SHA-256 over every row, ordered by primary key.
//
// Streamed rather than materialised: 877k rows by 22 columns would otherwise be
// a large amount of memory for no benefit.
func tableHash(db *sql.DB, cols []string) (string, error) {
	rows, err := db.Query("SELECT * FROM student ORDER BY so_bao_danh")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	h := sha256.New()
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		for i := range vals {
			io.WriteString(h, ser(vals[i]))
			io.WriteString(h, fieldSep)
		}
		io.WriteString(h, recordSep)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// firstDifferences walks both tables in step and reports where they diverge,
// so a failure names a row and a column rather than only a digest.
func firstDifferences(a, b *sql.DB, cols []string, limit int) ([]string, error) {
	ra, err := a.Query("SELECT * FROM student ORDER BY so_bao_danh")
	if err != nil {
		return nil, err
	}
	defer ra.Close()
	rb, err := b.Query("SELECT * FROM student ORDER BY so_bao_danh")
	if err != nil {
		return nil, err
	}
	defer rb.Close()

	scan := func(rows *sql.Rows) ([]string, bool, error) {
		if !rows.Next() {
			return nil, false, rows.Err()
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, false, err
		}
		out := make([]string, len(cols))
		for i := range vals {
			out[i] = ser(vals[i])
		}
		return out, true, nil
	}

	var out []string
	for len(out) < limit {
		va, okA, err := scan(ra)
		if err != nil {
			return nil, err
		}
		vb, okB, err := scan(rb)
		if err != nil {
			return nil, err
		}
		if !okA || !okB {
			break
		}
		for i, c := range cols {
			if va[i] != vb[i] {
				out = append(out, fmt.Sprintf("  so_bao_danh=%s %s: a=%q b=%q", va[0], c, va[i], vb[i]))
				break
			}
		}
	}
	return out, nil
}
