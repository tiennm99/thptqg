/// xlsxread — Rust CLI replacing the SheetJS xlsx build scripts.
///
/// Subcommands:
///   build  — read .xls/.xlsx files → write SQLite DB
///   audit  — compare distinct SBD count from xlsx vs DB row count
///
/// Library modules are declared in lib.rs; main.rs only adds the CLI layer.
mod cli;

use std::path::Path;

use anyhow::{Context, Result};
use clap::Parser;

use cli::{Cli, Cmd};
use xlsxread::audit;
use xlsxread::config::load_config;
use xlsxread::reader::{is_all_blank, process_file};
use xlsxread::transform::{validate_row, CompiledPatterns, SkipReason};
use xlsxread::writer::{finish_db, insert_row, open_db, SCORE_FIELDS};

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
// Build subcommand
// ---------------------------------------------------------------------------

fn run_build(schema_path: &Path, input_dir: &Path, output_path: &Path) -> Result<()> {
    let cfg = load_config(schema_path)
        .with_context(|| format!("Failed to load config: {}", schema_path.display()))?;

    // Compile score regexes once at startup
    let patterns =
        CompiledPatterns::new(&cfg.scores).with_context(|| "Failed to compile score regexes")?;

    // Collect input files (.xls and .xlsx), sorted for deterministic order
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

    // Open (or recreate) DB and apply DDL
    let conn = open_db(output_path, &cfg)
        .with_context(|| format!("Failed to open DB: {}", output_path.display()))?;

    let mut total_source_rows: u64 = 0;
    let mut total_skipped: u64 = 0;
    let mut total_errors: u64 = 0;

    let is_old2 = dataset_label.contains("old2");
    let strip_blank = cfg.reader.strip_blank_rows;

    // Single transaction over all files — mirrors the Node `db.transaction(() => { ... })()`
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

        let process_result = process_file(file, &cfg, |_sheet_idx, raw| {
            // data-old2: skip fully blank rows BEFORE counting sourceRows
            let all_blank = is_all_blank(raw);
            if strip_blank && all_blank {
                return;
            }

            total_source_rows += 1;

            let ho_ten = raw
                .get(cfg.columns.ho_ten)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();
            let so_bao_danh = raw
                .get(cfg.columns.so_bao_danh)
                .map(|c| c.to_string().trim().to_owned())
                .unwrap_or_default();

            match validate_row(
                &ho_ten,
                &so_bao_danh,
                &cfg.validation,
                strip_blank,
                all_blank,
            ) {
                Err(SkipReason::BlankRow) => {
                    // Already guarded above; won't reach here
                }
                Err(_) => {
                    file_skipped += 1;
                    return;
                }
                Ok(()) => {}
            }

            let parsed = xlsxread::transform::transform_row(raw, &cfg, &patterns);

            match insert_row(&conn, &cfg.insert.sql, &parsed, SCORE_FIELDS) {
                Ok(()) => {
                    file_rows += 1;
                }
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

        // Per-file row count line — mirrors `console.log(`  ${base}: ${fileRows} rows`)`
        println!("  {base}: {file_rows} rows");
    }

    conn.execute_batch("COMMIT")?;

    // VACUUM + stats output
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
