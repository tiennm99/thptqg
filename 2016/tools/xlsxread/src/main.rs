/// xlsxread — Rust CLI replacing the SheetJS xlsx build scripts.
///
/// Subcommands:
///   build  — read .xls/.xlsx files → write SQLite DB
///   audit  — compare distinct SBD count from xlsx vs DB row count
///
/// Library modules are declared in lib.rs; main.rs only adds the CLI layer.
///
/// When config contains `format_detection = "thptqg2016"` the build subcommand
/// uses per-file header inspection to pick the right column layout, replicating
/// the `detectFormat` logic from scripts/build-database.js (lines 63–87).
mod cli;

use std::path::Path;

use anyhow::{Context, Result};
use calamine::Data;
use clap::Parser;

use cli::{Cli, Cmd};
use xlsxread::audit;
use xlsxread::config::load_config;
use xlsxread::format_detect_2016::{
    detect_format, is_header_row_2016, process_row_2016, DetectedFormat,
};
use xlsxread::reader::{is_all_blank, process_file};
use xlsxread::transform::{validate_row, CompiledPatterns, SkipReason};
use xlsxread::writer::{
    finish_db, insert_row, insert_row_2016, open_db, SCORE_FIELDS, SCORE_FIELDS_2016,
};

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.cmd {
        Cmd::Build {
            schema,
            input,
            output,
        } => {
            run_build(&schema, &input, &output)?;
        }
        Cmd::Audit { schema, input, db } => {
            let cfg = load_config(&schema)
                .with_context(|| format!("Failed to load config: {}", schema.display()))?;
            let result = audit::run_audit(&input, &db, &cfg).with_context(|| "Audit failed")?;
            audit::print_audit_report(&result);
            if !result.matched {
                std::process::exit(1);
            }
        }
    }

    Ok(())
}

// ---------------------------------------------------------------------------
// Build subcommand — dispatches to thptqg2016 or standard path
// ---------------------------------------------------------------------------

fn run_build(schema_path: &Path, input_dir: &Path, output_path: &Path) -> Result<()> {
    let cfg = load_config(schema_path)
        .with_context(|| format!("Failed to load config: {}", schema_path.display()))?;

    if cfg.format_detection.as_deref() == Some("thptqg2016") {
        run_build_2016(&cfg, input_dir, output_path)
    } else {
        run_build_standard(&cfg, input_dir, output_path)
    }
}

// ---------------------------------------------------------------------------
// Standard build path (thptqg2017 and similar fixed-column configs)
// ---------------------------------------------------------------------------

fn run_build_standard(
    cfg: &xlsxread::config::DatasetConfig,
    input_dir: &Path,
    output_path: &Path,
) -> Result<()> {
    let patterns =
        CompiledPatterns::new(&cfg.scores).with_context(|| "Failed to compile score regexes")?;

    let mut files: Vec<std::path::PathBuf> = std::fs::read_dir(input_dir)
        .with_context(|| format!("Cannot read input dir: {}", input_dir.display()))?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.is_file()
                && p.extension()
                    .and_then(|e| e.to_str())
                    .map(|e| {
                        let lower = e.to_lowercase();
                        lower == "xls" || lower == "xlsx"
                    })
                    .unwrap_or(false)
        })
        .collect();
    files.sort();

    let dataset_label = input_dir
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("data");

    println!(
        "[build] {dataset_label}/ → {}  ({} files)",
        output_path.display(),
        files.len()
    );

    let conn = open_db(output_path, cfg)
        .with_context(|| format!("Failed to open DB: {}", output_path.display()))?;

    let mut total_source_rows: u64 = 0;
    let mut total_skipped: u64 = 0;
    let mut total_errors: u64 = 0;

    let is_old2 = dataset_label.contains("old2");
    let strip_blank = cfg.reader.strip_blank_rows;

    conn.execute_batch("BEGIN")?;

    for file in &files {
        let base = file
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("?")
            .to_owned();
        let mut file_rows: u64 = 0;
        let mut file_skipped: u64 = 0;
        let mut file_errors: u64 = 0;

        let process_result = process_file(file, cfg, |_sheet_idx, raw| {
            let all_blank = is_all_blank(raw);
            if strip_blank && all_blank {
                return;
            }

            total_source_rows += 1;

            let ho_ten = raw
                .get(cfg.columns.as_ref().unwrap().ho_ten)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();
            let so_bao_danh = raw
                .get(cfg.columns.as_ref().unwrap().so_bao_danh)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();

            match validate_row(&ho_ten, &so_bao_danh, &cfg.validation, strip_blank, all_blank) {
                Err(SkipReason::BlankRow) => {}
                Err(_) => {
                    file_skipped += 1;
                    return;
                }
                Ok(()) => {}
            }

            let parsed = xlsxread::transform::transform_row(raw, cfg, &patterns);

            match insert_row(&conn, &cfg.insert.sql, &parsed, SCORE_FIELDS) {
                Ok(()) => file_rows += 1,
                Err(e) => {
                    file_errors += 1;
                    if total_errors + file_errors <= 5 {
                        eprintln!("  [warn] {base}: {e}");
                    }
                }
            }
        });

        match process_result {
            Ok(_) => {}
            Err(e) => {
                eprintln!("  [error] {base}: {e}");
                file_errors += 1;
            }
        }

        total_skipped += file_skipped;
        total_errors += file_errors;
        println!("  {base}: {file_rows} rows");
    }

    conn.execute_batch("COMMIT")?;

    finish_db(
        &conn,
        output_path,
        total_source_rows,
        total_skipped,
        total_errors,
        dataset_label,
        files.len(),
        is_old2,
    )
    .with_context(|| "Failed to finalise DB")?;

    Ok(())
}

// ---------------------------------------------------------------------------
// thptqg2016 build path — per-file format detection
// ---------------------------------------------------------------------------

/// Build the thptqg2016 database.
///
/// Each file is processed independently: the first row is inspected to determine
/// which of the three column layouts applies (separate-scores / mapped / default).
/// This mirrors `detectFormat` in scripts/build-database.js lines 63–87, called
/// once per file inside the file loop at build-database.js:218–219.
fn run_build_2016(
    cfg: &xlsxread::config::DatasetConfig,
    input_dir: &Path,
    output_path: &Path,
) -> Result<()> {
    let patterns =
        CompiledPatterns::new(&cfg.scores).with_context(|| "Failed to compile score regexes")?;

    let mut files: Vec<std::path::PathBuf> = std::fs::read_dir(input_dir)
        .with_context(|| format!("Cannot read input dir: {}", input_dir.display()))?
        .filter_map(|e| e.ok())
        .map(|e| e.path())
        .filter(|p| {
            p.is_file()
                && p.extension()
                    .and_then(|e| e.to_str())
                    .map(|e| {
                        let lower = e.to_lowercase();
                        lower == "xls" || lower == "xlsx"
                    })
                    .unwrap_or(false)
        })
        .collect();
    files.sort();

    let dataset_label = input_dir
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("data");

    println!(
        "[build:2016] {dataset_label}/ → {}  ({} files)",
        output_path.display(),
        files.len()
    );

    let conn = open_db(output_path, cfg)
        .with_context(|| format!("Failed to open DB: {}", output_path.display()))?;

    let mut total_source_rows: u64 = 0;
    let total_skipped: u64 = 0;
    let mut total_errors: u64 = 0;

    conn.execute_batch("BEGIN")?;

    for file in &files {
        let base = file
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("?")
            .to_owned();

        match process_file_2016(
            file,
            cfg,
            &patterns,
            &conn,
            &base,
            &mut total_source_rows,
            &mut total_errors,
        ) {
            Ok(file_rows) => {
                println!("  {base}: {file_rows} rows");
            }
            Err(e) => {
                eprintln!("  [error] {base}: {e}");
                total_errors += 1;
            }
        }
    }

    conn.execute_batch("COMMIT")?;

    finish_db(
        &conn,
        output_path,
        total_source_rows,
        total_skipped,
        total_errors,
        dataset_label,
        files.len(),
        false,
    )
    .with_context(|| "Failed to finalise DB")?;

    Ok(())
}

/// Process one file in the thptqg2016 format-detection path.
///
/// Reads the file, uses the first row to detect the column layout, then processes
/// all subsequent data rows. Returns the count of successfully inserted rows.
fn process_file_2016(
    file: &Path,
    cfg: &xlsxread::config::DatasetConfig,
    patterns: &CompiledPatterns,
    conn: &rusqlite::Connection,
    base: &str,
    total_source_rows: &mut u64,
    total_errors: &mut u64,
) -> Result<u64> {
    use calamine::{open_workbook_auto, Reader, Sheets};

    let path_str = file.display().to_string();
    let mut workbook: Sheets<_> =
        open_workbook_auto(file).with_context(|| format!("Cannot open {path_str}"))?;

    let sheet_names: Vec<String> = workbook.sheet_names().to_vec();
    if sheet_names.is_empty() {
        return Ok(0);
    }

    // Sheet selection: thptqg2016 data/ has all-sheets mode to handle
    // HCM/HN overflow (same reason as thptqg2017 data/).
    let sheets_to_read: Vec<String> = match cfg.reader.sheet_mode {
        xlsxread::config::SheetMode::All => sheet_names.clone(),
        xlsxread::config::SheetMode::First => vec![sheet_names[0].clone()],
    };

    let mut file_rows: u64 = 0;

    for sheet_name in &sheets_to_read {
        let range = workbook
            .worksheet_range(sheet_name)
            .with_context(|| format!("Cannot read sheet {sheet_name} in {path_str}"))?;

        let rows: Vec<Vec<Data>> = range.rows().map(|r| r.to_vec()).collect();
        if rows.is_empty() {
            continue;
        }

        // Detect format from first row, then determine start index.
        // Mirrors build-database.js:215–220: isHeaderRow check + detectFormat.
        let (fmt, start_idx) = if is_header_row_2016(&rows[0]) {
            (detect_format(&rows[0]), 1)
        } else {
            (DetectedFormat::Default, 0)
        };

        for row in rows.iter().skip(start_idx) {
            if row.len() < 2 {
                continue;
            }

            *total_source_rows += 1;

            match process_row_2016(row, &fmt, patterns) {
                None => {
                    // Row was empty/invalid — skipped (mirrors JS `if (!record) continue`)
                }
                Some(parsed) => {
                    match insert_row_2016(conn, &cfg.insert.sql, &parsed, SCORE_FIELDS_2016) {
                        Ok(()) => file_rows += 1,
                        Err(e) => {
                            *total_errors += 1;
                            if *total_errors <= 5 {
                                eprintln!("  [warn] {base}: {e}");
                            }
                        }
                    }
                }
            }
        }
    }

    Ok(file_rows)
}
