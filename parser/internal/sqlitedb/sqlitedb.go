// Package sqlitedb registers the SQLite driver the parser writes with and names
// it in one place.
//
// modernc.org/sqlite is a pure-Go SQLite, chosen so the whole module stays
// cgo-free — grate, excelize and yaml.v3 are pure Go too, so CI can compile with
// CGO_ENABLED=0 and no C toolchain.
//
// It is a machine-transpiled SQLite rather than the upstream C amalgamation that
// Rust's rusqlite --bundled vendors (libsqlite3-sys 0.30.1). Two things make that
// acceptable: this parser uses only plain SQL — no CTEs, window functions,
// triggers or extensions — and the differential gate compares a full-table
// SHA-256 plus PRAGMA table_info/index_list against live Rust output.
//
// Verified on linux/arm64 with v1.56.0 (SQLite 3.53.3): the full DDL including
// the partial idx_ten_cum_thi index, INSERT OR REPLACE, and VACUUM.
//
// Fallback if the differential gate ever implicates the driver: mattn/go-sqlite3
// is upstream C at the cost of cgo.
package sqlitedb

// Registers "sqlite" with database/sql.
import _ "modernc.org/sqlite"

// DriverName is the database/sql driver name to pass to sql.Open.
const DriverName = "sqlite"
