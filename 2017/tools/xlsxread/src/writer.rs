/// SQLite writer: DDL setup, batched INSERT OR REPLACE, VACUUM, stats output.
///
/// Mirrors build-lib.js createDb + the transaction loop in each build-database*.js.
/// Stats output lines match the JS stdout exactly so existing CI log-greps still work.
use std::fs;
use std::path::Path;

use rusqlite::{params_from_iter, Connection, ToSql};

use crate::config::DatasetConfig;
use crate::error::BuildError;
use crate::transform::ParsedRow;

// ---------------------------------------------------------------------------
// DB initialisation — mirrors build-lib.js createDb (delete + recreate)
// ---------------------------------------------------------------------------

/// Open (or recreate) the output SQLite database, execute the DDL from config,
/// and return the open connection ready for inserts.
pub fn open_db(db_path: &Path, cfg: &DatasetConfig) -> Result<Connection, BuildError> {
    // Mirror Node behaviour: delete existing file before creating (build-lib.js:54)
    if db_path.exists() {
        fs::remove_file(db_path).map_err(|e| BuildError::Io {
            path: db_path.display().to_string(),
            source: e,
        })?;
    }

    // Ensure parent directory exists
    if let Some(parent) = db_path.parent() {
        if !parent.as_os_str().is_empty() {
            fs::create_dir_all(parent).map_err(|e| BuildError::Io {
                path: parent.display().to_string(),
                source: e,
            })?;
        }
    }

    let conn = Connection::open(db_path)?;
    conn.execute_batch(&cfg.schema.ddl)?;
    Ok(conn)
}

// ---------------------------------------------------------------------------
// Ordered score field list — canonical INSERT column order from build-lib.js
// ---------------------------------------------------------------------------

/// Fixed subject column order matching the INSERT statement in every config.
/// NULL is bound for any subject not present in a given row's score map.
pub const SCORE_FIELDS: &[&str] = &[
    "toan",
    "ngu_van",
    "vat_ly",
    "hoa_hoc",
    "sinh_hoc",
    "khtn",
    "lich_su",
    "dia_ly",
    "gdcd",
    "khxh",
    "tieng_anh",
    "tieng_phap",
    "tieng_nga",
    "tieng_trung",
];

// ---------------------------------------------------------------------------
// Insert a single parsed row inside an active transaction
// ---------------------------------------------------------------------------

/// Bind all fields from `row` into the prepared statement and execute it.
/// `score_fields` should be the ordered list of subject columns the INSERT expects.
pub fn insert_row(
    conn: &Connection,
    sql: &str,
    row: &ParsedRow,
    score_fields: &[&str],
) -> Result<(), BuildError> {
    // Build positional params: so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh, <scores...>
    let mut params: Vec<Box<dyn ToSql>> = Vec::with_capacity(4 + score_fields.len());
    params.push(Box::new(row.so_bao_danh.clone()));
    params.push(Box::new(row.ho_ten.clone()));
    params.push(Box::new(row.ho_ten_ascii.clone()));
    params.push(Box::new(row.ngay_sinh.clone()));

    for field in score_fields {
        let val: Option<f64> = row.scores.get(*field).copied();
        params.push(Box::new(val));
    }

    conn.execute(sql, params_from_iter(params.iter().map(|p| p.as_ref())))?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Post-build: VACUUM + stats output
// ---------------------------------------------------------------------------

/// Run VACUUM and print statistics lines that mirror the Node scripts' stdout.
/// The exact prefix tokens ("Source data rows", "DB rows", "Size:") are preserved
/// so any log-grep in the deploy pipeline keeps working.
#[allow(clippy::too_many_arguments)]
pub fn finish_db(
    conn: &Connection,
    db_path: &Path,
    source_rows: u64,
    skipped: u64,
    errors: u64,
    dataset_label: &str, // e.g. "data/" or "data-old2/"
    _file_count: usize,
    is_old2: bool, // data-old2 uses different label for the skipped line
) -> Result<(), BuildError> {
    conn.execute_batch("VACUUM")?;

    let db_count: i64 = conn.query_row("SELECT COUNT(*) FROM student", [], |row| row.get(0))?;

    let insertable = source_rows - skipped;

    // Mirror exact JS stdout format for each dataset variant
    println!();
    if is_old2 {
        println!("Source non-blank data rows:      {source_rows}");
        println!("  skipped (empty/non-numeric SBD): {skipped}");
    } else {
        println!("Source data rows (post-header):  {source_rows}");
        if dataset_label.contains("old") {
            println!("  skipped (empty/non-numeric SBD): {skipped}");
        } else {
            println!("  skipped (empty/invalid):        {skipped}");
        }
    }
    println!("  insertable:                     {insertable}");
    println!("  insert errors:                  {errors}");
    println!("DB rows (distinct SBD):           {db_count}");

    // Audit gap comment (mirrors build-database.js:80-83 for data/ only)
    if !dataset_label.contains("old") && errors == 0 {
        let gap = insertable as i64 - db_count;
        if gap == 0 {
            println!("Audit: OK — every source row made it in.");
        } else {
            println!("Audit: {gap} row(s) collapsed (duplicate SBDs overwriting).");
        }
    }

    let sz = fs::metadata(db_path).map(|m| m.len()).unwrap_or(0);
    println!("Size: {:.1} MB", sz as f64 / 1024.0 / 1024.0);

    Ok(())
}
