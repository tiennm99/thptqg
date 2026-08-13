//! Ground-truth cell dumper for the Go reader fidelity gate.
//!
//! Emits a canonical, reader-agnostic text rendering of every sheet and every
//! cell of a spreadsheet exactly as calamine sees it. The Go reader must
//! reproduce this byte-for-byte; the two dumps are compared by hash.
//!
//! Deliberately dumps the RAW used range: every sheet, every row, header rows
//! included, no config applied. This gate is about cell fidelity, not build
//! semantics — sheet selection and header skipping are exercised later.
//!
//! Usage: cargo run --release --example dump_cells -- <spreadsheet> [out-file]
//! With no out-file the dump goes to stdout.

use std::env;
use std::fs::File;
use std::io::{self, BufWriter, Write};

use calamine::{open_workbook_auto, Data, Reader, Sheets};

/// Escapes the field separators so a cell value can never break the line format.
fn escape(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for ch in s.chars() {
        match ch {
            '\\' => out.push_str("\\\\"),
            '\t' => out.push_str("\\t"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            _ => out.push(ch),
        }
    }
    out
}

/// Discriminates the calamine variant so a Go port can be checked against the
/// actual type, not just the rendered string.
fn kind(d: &Data) -> &'static str {
    match d {
        Data::Empty => "empty",
        Data::String(_) => "str",
        Data::Float(_) => "float",
        Data::Int(_) => "int",
        Data::Bool(_) => "bool",
        Data::Error(_) => "err",
        Data::DateTime(_) => "datetime",
        Data::DateTimeIso(_) => "datetimeiso",
        Data::DurationIso(_) => "durationiso",
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        eprintln!("usage: dump_cells <spreadsheet> [out-file]");
        std::process::exit(2);
    }
    let path = &args[1];

    let mut out: Box<dyn Write> = match args.get(2) {
        Some(p) => Box::new(BufWriter::new(File::create(p)?)),
        None => Box::new(BufWriter::new(io::stdout())),
    };

    let mut workbook: Sheets<_> = open_workbook_auto(path)?;
    let sheet_names: Vec<String> = workbook.sheet_names().to_vec();

    writeln!(out, "FILE\t{}", escape(path))?;
    writeln!(out, "SHEETCOUNT\t{}", sheet_names.len())?;

    for (idx, name) in sheet_names.iter().enumerate() {
        let range = workbook.worksheet_range(name)?;
        // start() is the used-range origin — the key question for any Go reader,
        // which may index absolutely from A1 instead.
        let (srow, scol) = range.start().unwrap_or((0, 0));
        writeln!(
            out,
            "SHEET\t{}\t{}\t{}\t{}\t{}\t{}",
            idx,
            escape(name),
            range.height(),
            range.width(),
            srow,
            scol
        )?;

        for (r, row) in range.rows().enumerate() {
            writeln!(out, "ROW\t{}\t{}\t{}", idx, r, row.len())?;
            for (c, cell) in row.iter().enumerate() {
                let s = cell.to_string();
                writeln!(
                    out,
                    "CELL\t{}\t{}\t{}\t{}\t{}\t{}",
                    idx,
                    r,
                    c,
                    kind(cell),
                    if matches!(cell, Data::Empty) { 1 } else { 0 },
                    escape(&s)
                )?;
            }
        }
    }

    out.flush()?;
    Ok(())
}
