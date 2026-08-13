//! Corpus-wide cell-kind and sheet-geometry scan, for the Go reader fidelity gate.
//!
//! For every file given, prints one TSV line per sheet and one summary line per
//! file. Aggregates only — never materialises the cell text — so the whole
//! 418 MB corpus can be scanned quickly.
//!
//! The point is to find out which calamine `Data` variants actually occur in
//! real inputs. `DateTime` and `Float` are the variants whose rendering differs
//! between readers; if they never appear, the divergence risk is theoretical.
//!
//! Usage: cargo run --release --example scan_kinds -- <file>...

use std::env;

use calamine::{open_workbook_auto, Data, Reader, Sheets};

fn main() {
    let args: Vec<String> = env::args().skip(1).collect();
    if args.is_empty() {
        eprintln!("usage: scan_kinds <file>...");
        std::process::exit(2);
    }

    println!("#TYPE\tpath\tsheet_idx\tname\theight\twidth\tstart_row\tstart_col\tempty\tstr\tfloat\tint\tbool\tdatetime\tdtiso\tduriso\terr");

    for path in &args {
        let mut wb: Sheets<_> = match open_workbook_auto(path) {
            Ok(w) => w,
            Err(e) => {
                println!("ERR\t{path}\t{e}");
                continue;
            }
        };

        let names: Vec<String> = wb.sheet_names().to_vec();
        for (idx, name) in names.iter().enumerate() {
            let range = match wb.worksheet_range(name) {
                Ok(r) => r,
                Err(e) => {
                    println!("SHEETERR\t{path}\t{idx}\t{e}");
                    continue;
                }
            };
            let (sr, sc) = range.start().unwrap_or((0, 0));
            let mut k = [0usize; 9]; // empty,str,float,int,bool,datetime,dtiso,duriso,err
            for row in range.rows() {
                for cell in row {
                    let i = match cell {
                        Data::Empty => 0,
                        Data::String(_) => 1,
                        Data::Float(_) => 2,
                        Data::Int(_) => 3,
                        Data::Bool(_) => 4,
                        Data::DateTime(_) => 5,
                        Data::DateTimeIso(_) => 6,
                        Data::DurationIso(_) => 7,
                        Data::Error(_) => 8,
                    };
                    k[i] += 1;
                }
            }
            println!(
                "SHEET\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}\t{}",
                path,
                idx,
                name.replace('\t', " "),
                range.height(),
                range.width(),
                sr,
                sc,
                k[0], k[1], k[2], k[3], k[4], k[5], k[6], k[7], k[8]
            );
        }
        println!("FILE\t{}\t{}", path, names.len());
    }
}
