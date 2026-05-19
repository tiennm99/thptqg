/// SQLite writer: DDL setup, batched INSERT OR REPLACE, VACUUM, stats output.
///
/// Mirrors build-lib.js createDb + the transaction loop in each build-database*.js.
/// Stats output lines match the JS stdout exactly so existing CI log-greps still work.
///
/// thptqg2016 differences vs thptqg2017:
///   - INSERT includes ten_cum_thi and gioi_tinh after ngay_sinh
///   - Score columns are toan/ngu_van/vat_ly/hoa_hoc/sinh_hoc/lich_su/dia_ly/
///     tieng_anh/tieng_phap/tieng_duc/tieng_nhat/tieng_trung
///     (no khtn, khxh, tieng_nga)
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
// Score field lists — one per dataset schema
// ---------------------------------------------------------------------------

/// thptqg2017 score columns (14 fields including khtn/khxh/tieng_nga).
pub const SCORE_FIELDS_2017: &[&str] = &[
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

/// thptqg2016 score columns (12 fields: tieng_duc/tieng_nhat present; no khtn/khxh/tieng_nga).
pub const SCORE_FIELDS_2016: &[&str] = &[
    "toan",
    "ngu_van",
    "vat_ly",
    "hoa_hoc",
    "sinh_hoc",
    "lich_su",
    "dia_ly",
    "tieng_anh",
    "tieng_phap",
    "tieng_duc",
    "tieng_nhat",
    "tieng_trung",
];

/// Alias kept so thptqg2017 callers that import SCORE_FIELDS continue to compile.
pub const SCORE_FIELDS: &[&str] = SCORE_FIELDS_2017;

// ---------------------------------------------------------------------------
// Insert a single parsed row (thptqg2017 schema — no ten_cum_thi / gioi_tinh)
// ---------------------------------------------------------------------------

/// Bind all fields from `row` into the prepared statement and execute it.
/// Positional params: so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh, <scores...>
pub fn insert_row(
    conn: &Connection,
    sql: &str,
    row: &ParsedRow,
    score_fields: &[&str],
) -> Result<(), BuildError> {
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
// Insert a single parsed row (thptqg2016 schema — includes ten_cum_thi + gioi_tinh)
// ---------------------------------------------------------------------------

/// Bind all fields from `row` into the prepared statement for the thptqg2016 schema.
/// Positional params: so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh, ten_cum_thi,
///                    gioi_tinh, <scores...>
pub fn insert_row_2016(
    conn: &Connection,
    sql: &str,
    row: &ParsedRow,
    score_fields: &[&str],
) -> Result<(), BuildError> {
    let mut params: Vec<Box<dyn ToSql>> = Vec::with_capacity(6 + score_fields.len());
    params.push(Box::new(row.so_bao_danh.clone()));
    params.push(Box::new(row.ho_ten.clone()));
    params.push(Box::new(row.ho_ten_ascii.clone()));
    params.push(Box::new(row.ngay_sinh.clone()));
    params.push(Box::new(row.ten_cum_thi.clone()));
    params.push(Box::new(row.gioi_tinh.clone()));

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
    dataset_label: &str,
    _file_count: usize,
    is_old2: bool,
) -> Result<(), BuildError> {
    conn.execute_batch("VACUUM")?;

    let db_count: i64 = conn.query_row("SELECT COUNT(*) FROM student", [], |row| row.get(0))?;

    let insertable = source_rows - skipped;

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
