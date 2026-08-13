/// Spreadsheet reader: wraps calamine to iterate rows across sheets.
///
/// Sheet selection mirrors the JS scripts:
///   - sheet_mode = "all"   → iterate every sheet (handles HCM/HN 65k overflow in data/)
///   - sheet_mode = "first" → sheet 0 only (data-old/)
///
/// Header detection mirrors build-lib.js isHeaderRow:
///   row[0].toUpperCase() in {"HO_TEN", "HỌ TÊN", "STT"}
use std::path::Path;

use calamine::{open_workbook_auto, Data, Reader, Sheets};

use crate::config::{DatasetConfig, HeaderCfg, SheetMode};
use crate::error::BuildError;

// ---------------------------------------------------------------------------
// Public row representation from calamine
// ---------------------------------------------------------------------------

pub type RawRow = Vec<Data>;

// ---------------------------------------------------------------------------
// Header detection — mirrors build-lib.js isHeaderRow
// ---------------------------------------------------------------------------

/// Returns true when the first cell (uppercased) matches one of the configured
/// header tokens. Used to skip the header row on the first row of each sheet.
pub fn is_header_row(row: &[Data], header_cfg: &HeaderCfg) -> bool {
    if row.len() < 3 {
        return false;
    }
    let first = row[0].to_string().trim().to_uppercase();
    header_cfg.tokens.iter().any(|t| t.to_uppercase() == first)
}

// ---------------------------------------------------------------------------
// All-blank row check (data-old2: strip_blank_rows)
// ---------------------------------------------------------------------------

pub fn is_all_blank(row: &[Data]) -> bool {
    row.iter()
        .all(|c| matches!(c, Data::Empty) || c.to_string().trim().is_empty())
}

// ---------------------------------------------------------------------------
// File processor — yields all data rows from the file
// ---------------------------------------------------------------------------

/// Process one spreadsheet file, calling `on_row` for each data row.
///
/// `on_row` receives `(sheet_index, row_index_in_sheet, raw_row)` where
/// `row_index_in_sheet` is 0-based AFTER the header has been consumed.
/// Returns `(sheets_seen, total_rows_yielded)`.
pub fn process_file<F>(
    path: &Path,
    cfg: &DatasetConfig,
    mut on_row: F,
) -> Result<(usize, usize), BuildError>
where
    F: FnMut(usize, &RawRow),
{
    let path_str = path.display().to_string();

    // calamine::open_workbook_auto dispatches on file extension
    let mut workbook: Sheets<_> = open_workbook_auto(path).map_err(|e| BuildError::Calamine {
        path: path_str.clone(),
        source: e,
    })?;

    let sheet_names: Vec<String> = workbook.sheet_names().to_vec();
    if sheet_names.is_empty() {
        return Err(BuildError::NoSheets(path_str.clone()));
    }

    // Sheet selection per config
    let sheets_to_read: Vec<String> = match cfg.reader.sheet_mode {
        SheetMode::All => sheet_names.clone(),
        SheetMode::First => vec![sheet_names[0].clone()],
    };

    let mut total_rows = 0usize;

    for (sheet_idx, sheet_name) in sheets_to_read.iter().enumerate() {
        let range = workbook
            .worksheet_range(sheet_name)
            .map_err(|e| BuildError::Calamine {
                path: path_str.clone(),
                source: e,
            })?;

        let mut first_row = true;

        for raw in range.rows() {
            let row: RawRow = raw.to_vec();

            // Skip header row on first row of each sheet (matches JS: `if (i === 0 && isHeaderRow(...))`)
            if first_row {
                first_row = false;
                if is_header_row(&row, &cfg.header) {
                    continue;
                }
            }

            on_row(sheet_idx, &row);
            total_rows += 1;
        }
    }

    Ok((sheets_to_read.len(), total_rows))
}

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::HeaderCfg;

    fn hdr(tokens: &[&str]) -> HeaderCfg {
        HeaderCfg {
            tokens: tokens.iter().map(|s| s.to_string()).collect(),
        }
    }

    #[test]
    fn header_detects_ho_ten() {
        let row = vec![
            Data::String("HO_TEN".into()),
            Data::String("NGAY_SINH".into()),
            Data::String("SBD".into()),
        ];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(is_header_row(&row, &cfg));
    }

    #[test]
    fn header_detects_stt() {
        let row = vec![
            Data::String("STT".into()),
            Data::String("B".into()),
            Data::String("C".into()),
        ];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(is_header_row(&row, &cfg));
    }

    #[test]
    fn header_detects_ho_ten_unicode() {
        let row = vec![
            Data::String("HỌ TÊN".into()),
            Data::String("B".into()),
            Data::String("C".into()),
        ];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(is_header_row(&row, &cfg));
    }

    #[test]
    fn header_rejects_data_row() {
        let row = vec![
            Data::String("Nguyen Van A".into()),
            Data::String("01/01/2000".into()),
            Data::String("12345678".into()),
        ];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(!is_header_row(&row, &cfg));
    }

    #[test]
    fn header_rejects_short_row() {
        let row = vec![Data::String("HO_TEN".into()), Data::Empty];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(!is_header_row(&row, &cfg));
    }

    #[test]
    fn header_case_insensitive() {
        let row = vec![
            Data::String("ho_ten".into()),
            Data::String("B".into()),
            Data::String("C".into()),
        ];
        let cfg = hdr(&["HO_TEN", "HỌ TÊN", "STT"]);
        assert!(is_header_row(&row, &cfg));
    }

    #[test]
    fn blank_row_detection() {
        let row = vec![Data::Empty, Data::Empty, Data::String("".into())];
        assert!(is_all_blank(&row));

        let row2 = vec![Data::String("Nguyen".into()), Data::Empty, Data::Empty];
        assert!(!is_all_blank(&row2));
    }
}
