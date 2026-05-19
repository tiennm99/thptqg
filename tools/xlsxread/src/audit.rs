/// Audit subcommand: replicates audit-row-counts.js exactly.
///
/// Reads all .xlsx files from the input directory (sheet 0 only, matching the
/// JS script's behaviour at audit-row-counts.js:33), collects distinct SBDs
/// into a HashSet, then queries `SELECT COUNT(*) FROM student` from the DB.
/// Prints the same lines as audit-row-counts.js:54-62 and exits 0 on match,
/// 1 on mismatch.
use std::collections::HashSet;
use std::path::Path;

use calamine::{open_workbook_auto, Data, Reader};

use crate::config::DatasetConfig;
use crate::error::BuildError;
use crate::reader::is_header_row;

// ---------------------------------------------------------------------------
// Audit result
// ---------------------------------------------------------------------------

pub struct AuditResult {
    pub total_data_rows: u64,
    pub both_empty: u64,
    pub empty_name: u64,
    pub empty_sbd: u64,
    pub distinct_sbds: usize,
    pub db_count: i64,
    pub matched: bool,
}

// ---------------------------------------------------------------------------
// Main audit logic
// ---------------------------------------------------------------------------

/// Collect distinct SBDs from all xlsx files in `input_dir`, query `db_path`,
/// print the audit report and return the result.
///
/// The JS script reads only sheet 0 for every file (audit-row-counts.js:33).
/// Unlike build-database.js, the audit script does NOT iterate all sheets.
pub fn run_audit(
    input_dir: &Path,
    db_path: &Path,
    cfg: &DatasetConfig,
) -> Result<AuditResult, BuildError> {
    // Collect .xlsx files (audit-row-counts.js only checks .xlsx — line 15)
    let mut files: Vec<std::path::PathBuf> = std::fs::read_dir(input_dir)
        .map_err(|e| BuildError::Io {
            path: input_dir.display().to_string(),
            source: e,
        })?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.is_file()
                && p.extension()
                    .and_then(|e| e.to_str())
                    .map(|e| e.eq_ignore_ascii_case("xlsx"))
                    .unwrap_or(false)
        })
        .collect();
    files.sort();

    let mut all_sbd: HashSet<String> = HashSet::new();
    let mut total_data_rows: u64 = 0;
    let mut empty_name: u64 = 0;
    let mut empty_sbd: u64 = 0;
    let mut both_empty: u64 = 0;

    for file in &files {
        let path_str = file.display().to_string();
        let mut workbook = open_workbook_auto(file).map_err(|e| BuildError::Calamine {
            path: path_str.clone(),
            source: e,
        })?;

        let sheet_names = workbook.sheet_names().to_vec();
        if sheet_names.is_empty() {
            continue;
        }

        // audit-row-counts.js reads only sheet 0 (line 33: wb.SheetNames[0])
        let range =
            workbook
                .worksheet_range(&sheet_names[0])
                .map_err(|e| BuildError::Calamine {
                    path: path_str.clone(),
                    source: e,
                })?;

        let mut first_row = true;
        for raw in range.rows() {
            let row: Vec<Data> = raw.to_vec();

            // Skip header row on first row only
            if first_row {
                first_row = false;
                if is_header_row(&row, &cfg.header) {
                    continue;
                }
            }

            total_data_rows += 1;

            let ho_ten = row
                .get(cfg.columns.ho_ten)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();
            let sbd = row
                .get(cfg.columns.so_bao_danh)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();

            if ho_ten.is_empty() && sbd.is_empty() {
                both_empty += 1;
                continue;
            }
            if ho_ten.is_empty() {
                empty_name += 1;
            }
            if sbd.is_empty() {
                empty_sbd += 1;
            }
            if !sbd.is_empty() {
                all_sbd.insert(sbd);
            }
        }
    }

    // Query DB count
    let conn =
        rusqlite::Connection::open_with_flags(db_path, rusqlite::OpenFlags::SQLITE_OPEN_READ_ONLY)?;
    let db_count: i64 = conn.query_row("SELECT COUNT(*) FROM student", [], |row| row.get(0))?;

    let distinct_sbds = all_sbd.len();
    let matched = distinct_sbds as i64 == db_count;

    Ok(AuditResult {
        total_data_rows,
        both_empty,
        empty_name,
        empty_sbd,
        distinct_sbds,
        db_count,
        matched,
    })
}

// ---------------------------------------------------------------------------
// Print audit report — mirrors audit-row-counts.js:54-62 exactly
// ---------------------------------------------------------------------------

pub fn print_audit_report(r: &AuditResult) {
    println!("=== Source vs DB ===");
    println!(
        "Source: total data rows across all files: {}",
        r.total_data_rows
    );
    println!(
        "Source: rows with empty name AND sbd (skipped): {}",
        r.both_empty
    );
    println!("Source: rows with missing name only: {}", r.empty_name);
    println!("Source: rows with missing sbd only: {}", r.empty_sbd);
    println!("Source: distinct SBDs: {}", r.distinct_sbds);
    println!("DB:     row count: {}", r.db_count);
    println!(
        "Match:  {}",
        if r.matched {
            "YES — all unique SBDs accounted for".to_string()
        } else {
            format!("NO — gap of {}", r.distinct_sbds as i64 - r.db_count)
        }
    );
}
