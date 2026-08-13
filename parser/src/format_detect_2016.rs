/// Per-file format auto-detection for the thptqg2016 dataset.
///
/// Translates `detectFormat` from scripts/build-database.js (lines 63–87) and the
/// three row-processing functions (lines 90–146) into Rust.
///
/// The JS source has three formats:
///
/// 1. `separate-scores`  — header row[0]=="SBD" && row[2]=="TOAN"
///    Columns: SBD(0) HOTEN(1) TOAN(2) VAN(3) LY(4) HOA(5) SINH(6) SU(7) DIA(8)
///    NGOAINGUTN(9) NGOAINGUTL(10) NGOAINGU-total(11)
///    → maps col 11 → tieng_anh; no ngay_sinh / ten_cum_thi / gioi_tinh / DIEM_THI
///    → JS: build-database.js:90–116  (processSeparateScoresRow)
///
/// 2. `mapped`           — header row has SOBAODANH|SBD and DIEM_THI columns
///    → dynamic column indices built from header names
///    → JS: build-database.js:119–146 (processMappedRow with map from detectFormat)
///
/// 3. `default`          — no recognised header; positional 6-col layout
///    SBD(0) HO_TEN(1) NGAY_SINH(2) TEN_CUMTHI(3) GIOI_TINH(4) DIEM_THI(5)
///    → JS: build-database.js:149–151 DEFAULT_MAP + processMappedRow
///
/// JS citations are line numbers in /config/workspace/tiennm99/thptqg2016/scripts/build-database.js.
use calamine::Data;

use crate::transform::{parse_scores, to_ascii, CompiledPatterns, ParsedRow};

// ---------------------------------------------------------------------------
// Known header tokens (mirrors JS KNOWN_HEADERS set, build-database.js:50–54)
// ---------------------------------------------------------------------------

/// Upper-cased strings that identify a header row's first cell.
/// Mirrors `KNOWN_HEADERS` in build-database.js:50–54.
const KNOWN_HEADERS: &[&str] = &[
    "SOBAODANH",
    "SBD",
    "HO_TEN",
    "HOTEN",
    "HỌ TÊN",
    "NGAY_SINH",
    "TEN_CUMTHI",
    "GIOI_TINH",
    "DIEM_THI",
    "STT",
    "TOAN",
    "VAN",
    "LY",
    "HOA",
    "SINH ",
    "SU",
    "DIA",
];

/// Returns true when `row[0]` (uppercased, trimmed) is in the known-headers set.
/// Mirrors `isHeaderRow` at build-database.js:56–60.
pub fn is_header_row_2016(row: &[Data]) -> bool {
    if row.len() < 2 {
        return false;
    }
    let first = row[0].to_string().trim().to_uppercase();
    KNOWN_HEADERS.contains(&first.as_str())
}

// ---------------------------------------------------------------------------
// Detected per-file format
// ---------------------------------------------------------------------------

/// The three layouts the thptqg2016 dataset uses, detected per file.
#[derive(Debug, Clone)]
pub enum DetectedFormat {
    /// SBD/HOTEN/TOAN/VAN/LY/HOA/SINH/SU/DIA/NGOAINGUTN/NGOAINGUTL/NGOAINGU columns.
    /// Corresponds to the dhhanghai-style files. build-database.js:67–68.
    SeparateScores,
    /// Header present with SOBAODANH|SBD and DIEM_THI; dynamic column indices.
    /// build-database.js:70–86.
    Mapped {
        sbd: usize,
        ho_ten: usize,
        ngay_sinh: Option<usize>,
        ten_cum_thi: Option<usize>,
        gioi_tinh: Option<usize>,
        diem_thi: usize,
    },
    /// No recognised header; standard 6-column positional layout.
    /// build-database.js:149–151 DEFAULT_MAP.
    Default,
}

/// Inspect a header row and decide which format applies.
/// Returns `None` when the header is present but unrecognised (treated as Default).
///
/// Mirrors `detectFormat` at build-database.js:63–87.
pub fn detect_format(header_row: &[Data]) -> DetectedFormat {
    let cols: Vec<String> = header_row
        .iter()
        .map(|c| c.to_string().trim().to_uppercase())
        .collect();

    // Format 1: SBD in col 0 AND TOAN in col 2 → separate-scores
    // build-database.js:68: if (cols[0] === "SBD" && cols[2] === "TOAN")
    if cols.first().map(|s| s.as_str()) == Some("SBD")
        && cols.get(2).map(|s| s.as_str()) == Some("TOAN")
    {
        return DetectedFormat::SeparateScores;
    }

    // Format 2: build column index map — check for SOBAODANH|SBD and DIEM_THI
    // build-database.js:70–86
    let mut sbd_idx: Option<usize> = None;
    let mut ho_ten_idx: Option<usize> = None;
    let mut ngay_sinh_idx: Option<usize> = None;
    let mut ten_cum_thi_idx: Option<usize> = None;
    let mut gioi_tinh_idx: Option<usize> = None;
    let mut diem_thi_idx: Option<usize> = None;

    for (i, c) in cols.iter().enumerate() {
        match c.as_str() {
            "SOBAODANH" | "SBD" => sbd_idx = Some(i),
            "HO_TEN" | "HOTEN" | "HỌ TÊN" => ho_ten_idx = Some(i),
            "NGAY_SINH" => ngay_sinh_idx = Some(i),
            "TEN_CUMTHI" => ten_cum_thi_idx = Some(i),
            "GIOI_TINH" => gioi_tinh_idx = Some(i),
            "DIEM_THI" => diem_thi_idx = Some(i),
            _ => {}
        }
    }

    // build-database.js:82–84: if (map.sbd !== undefined && map.diem_thi !== undefined)
    if let (Some(sbd), Some(diem_thi)) = (sbd_idx, diem_thi_idx) {
        let ho_ten = ho_ten_idx.unwrap_or(1); // fallback: col 1 (present in all known files)
        return DetectedFormat::Mapped {
            sbd,
            ho_ten,
            ngay_sinh: ngay_sinh_idx,
            ten_cum_thi: ten_cum_thi_idx,
            gioi_tinh: gioi_tinh_idx,
            diem_thi,
        };
    }

    // Unrecognised header (or no header) → positional default
    DetectedFormat::Default
}

// ---------------------------------------------------------------------------
// Row processors
// ---------------------------------------------------------------------------

/// Cell accessor helper.
fn cell_str(row: &[Data], idx: usize) -> String {
    row.get(idx)
        .map(|c| c.to_string().trim().to_owned())
        .unwrap_or_default()
}

/// Parse a cell that should hold a float score; returns None for blank/non-numeric.
/// Mirrors `parseFloat(row[N]) || null` in JS.
fn parse_float_cell(row: &[Data], idx: usize) -> Option<f64> {
    let s = cell_str(row, idx);
    if s.is_empty() {
        return None;
    }
    s.parse::<f64>().ok().filter(|v| v.is_finite() && *v != 0.0)
}

/// Process a row in the `separate-scores` format.
///
/// Column layout (build-database.js:90–116 `processSeparateScoresRow`):
///   0=SBD  1=HOTEN  2=TOAN  3=VAN  4=LY  5=HOA  6=SINH  7=SU  8=DIA
///   9=NGOAINGUTN  10=NGOAINGUTL  11=NGOAINGU(total→tieng_anh)
///
/// tieng_phap / tieng_duc / tieng_nhat / tieng_trung all → None
/// ngay_sinh / ten_cum_thi / gioi_tinh all → None (not in this format)
pub fn process_separate_scores_row(row: &[Data], patterns: &CompiledPatterns) -> Option<ParsedRow> {
    let sbd = cell_str(row, 0);
    let ho_ten = cell_str(row, 1);
    if sbd.is_empty() || ho_ten.is_empty() {
        return None;
    }

    let ho_ten_ascii = to_ascii(&ho_ten);

    // build-database.js:102–114: explicit per-column score mapping
    let mut scores = std::collections::HashMap::new();
    macro_rules! add_score {
        ($field:expr, $idx:expr) => {
            if let Some(v) = parse_float_cell(row, $idx) {
                scores.insert($field.to_string(), v);
            }
        };
    }
    add_score!("toan", 2);
    add_score!("ngu_van", 3);
    add_score!("vat_ly", 4);
    add_score!("hoa_hoc", 5);
    add_score!("sinh_hoc", 6);
    add_score!("lich_su", 7);
    add_score!("dia_ly", 8);
    // col 11 = NGOAINGU total → tieng_anh (build-database.js:110–111)
    add_score!("tieng_anh", 11);

    // Suppress unused-variable warning; patterns not used in this path (no DIEM_THI string)
    let _ = patterns;

    Some(ParsedRow {
        so_bao_danh: sbd,
        ho_ten,
        ho_ten_ascii,
        ngay_sinh: None,
        ten_cum_thi: None,
        gioi_tinh: None,
        scores,
    })
}

/// Process a row in the `mapped` format (header-derived column indices).
///
/// Mirrors `processMappedRow` at build-database.js:119–146.
/// Gender is normalised: only "Nam" or "Nữ" are kept; everything else → None.
/// (build-database.js:132: `(rawGioiTinh === "Nam" || rawGioiTinh === "Nữ") ? rawGioiTinh : null`)
// Mirrors the JS column map one-for-one; grouping the indices into a struct
// would obscure that correspondence for no benefit.
#[allow(clippy::too_many_arguments)]
pub fn process_mapped_row(
    row: &[Data],
    sbd_idx: usize,
    ho_ten_idx: usize,
    ngay_sinh_idx: Option<usize>,
    ten_cum_thi_idx: Option<usize>,
    gioi_tinh_idx: Option<usize>,
    diem_thi_idx: usize,
    patterns: &CompiledPatterns,
) -> Option<ParsedRow> {
    let sbd = cell_str(row, sbd_idx);
    let ho_ten = cell_str(row, ho_ten_idx);
    if sbd.is_empty() || ho_ten.is_empty() {
        return None;
    }

    // Skip rows where SBD or HO_TEN are themselves header tokens (leaked header rows).
    // build-database.js:125–126: KNOWN_HEADERS.has(sbdUpper) || KNOWN_HEADERS.has(hoTenUpper)
    let sbd_upper = sbd.to_uppercase();
    let ho_ten_upper = ho_ten.to_uppercase();
    if KNOWN_HEADERS.contains(&sbd_upper.as_str())
        || KNOWN_HEADERS.contains(&ho_ten_upper.as_str())
    {
        return None;
    }

    let ho_ten_ascii = to_ascii(&ho_ten);

    let ngay_sinh = ngay_sinh_idx
        .map(|i| cell_str(row, i))
        .filter(|s| !s.is_empty());

    let ten_cum_thi = ten_cum_thi_idx
        .map(|i| cell_str(row, i))
        .filter(|s| !s.is_empty());

    // build-database.js:130–132: normalise gender
    let gioi_tinh = gioi_tinh_idx
        .map(|i| cell_str(row, i))
        .and_then(|s| {
            if s == "Nam" || s == "Nữ" {
                Some(s)
            } else {
                None
            }
        });

    let diem_thi = row
        .get(diem_thi_idx)
        .map(|c| c.to_string())
        .unwrap_or_default();

    let scores = parse_scores(&diem_thi, patterns);

    Some(ParsedRow {
        so_bao_danh: sbd,
        ho_ten,
        ho_ten_ascii,
        ngay_sinh,
        ten_cum_thi,
        gioi_tinh,
        scores,
    })
}

/// Process a row using the default positional 6-column layout.
///
/// Column order: SBD(0) HO_TEN(1) NGAY_SINH(2) TEN_CUMTHI(3) GIOI_TINH(4) DIEM_THI(5)
/// Mirrors `processMappedRow(row, DEFAULT_MAP)` at build-database.js:233–236.
pub fn process_default_row(row: &[Data], patterns: &CompiledPatterns) -> Option<ParsedRow> {
    process_mapped_row(
        row,
        0, // sbd
        1, // ho_ten
        Some(2),
        Some(3),
        Some(4),
        5, // diem_thi
        patterns,
    )
}

/// Dispatch a data row through the correct processor for the detected format.
///
/// Returns `None` when the row is empty/invalid and should be skipped.
pub fn process_row_2016(
    row: &[Data],
    fmt: &DetectedFormat,
    patterns: &CompiledPatterns,
) -> Option<ParsedRow> {
    match fmt {
        DetectedFormat::SeparateScores => process_separate_scores_row(row, patterns),
        DetectedFormat::Mapped {
            sbd,
            ho_ten,
            ngay_sinh,
            ten_cum_thi,
            gioi_tinh,
            diem_thi,
        } => process_mapped_row(
            row,
            *sbd,
            *ho_ten,
            *ngay_sinh,
            *ten_cum_thi,
            *gioi_tinh,
            *diem_thi,
            patterns,
        ),
        DetectedFormat::Default => process_default_row(row, patterns),
    }
}

// ---------------------------------------------------------------------------
// Unit tests — 3 detection branches + key processing cases
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    fn s(v: &str) -> Data {
        Data::String(v.to_string())
    }

    fn make_patterns() -> CompiledPatterns {
        CompiledPatterns::new().unwrap()
    }

    // --- detect_format: branch 1 — separate-scores ---

    #[test]
    fn detect_separate_scores() {
        let header = vec![s("SBD"), s("HOTEN"), s("TOAN"), s("VAN")];
        match detect_format(&header) {
            DetectedFormat::SeparateScores => {}
            other => panic!("expected SeparateScores, got {other:?}"),
        }
    }

    // --- detect_format: branch 2 — mapped ---

    #[test]
    fn detect_mapped_with_named_cols() {
        let header = vec![
            s("SOBAODANH"),
            s("HO_TEN"),
            s("NGAY_SINH"),
            s("TEN_CUMTHI"),
            s("GIOI_TINH"),
            s("DIEM_THI"),
        ];
        match detect_format(&header) {
            DetectedFormat::Mapped {
                sbd,
                ho_ten,
                ngay_sinh,
                ten_cum_thi,
                gioi_tinh,
                diem_thi,
            } => {
                assert_eq!(sbd, 0);
                assert_eq!(ho_ten, 1);
                assert_eq!(ngay_sinh, Some(2));
                assert_eq!(ten_cum_thi, Some(3));
                assert_eq!(gioi_tinh, Some(4));
                assert_eq!(diem_thi, 5);
            }
            other => panic!("expected Mapped, got {other:?}"),
        }
    }

    #[test]
    fn detect_mapped_sbd_variant() {
        // "SBD" (not "SOBAODANH") + DIEM_THI at different positions
        let header = vec![s("STT"), s("SBD"), s("HOTEN"), s("DIEM_THI")];
        match detect_format(&header) {
            DetectedFormat::Mapped { sbd, diem_thi, .. } => {
                assert_eq!(sbd, 1);
                assert_eq!(diem_thi, 3);
            }
            other => panic!("expected Mapped, got {other:?}"),
        }
    }

    // --- detect_format: branch 3 — default ---

    #[test]
    fn detect_default_when_no_header() {
        // A data row: positional default used
        let data_row = vec![
            s("12345678"),
            s("Nguyễn Văn A"),
            s("01/01/2000"),
            s("TP HCM"),
            s("Nam"),
            s("Toán: 8.5"),
        ];
        // Default is returned when there is no recognised header
        match detect_format(&data_row) {
            DetectedFormat::Default => {}
            other => panic!("expected Default, got {other:?}"),
        }
    }

    // --- process_separate_scores_row ---

    #[test]
    fn separate_scores_basic() {
        let p = make_patterns();
        // 12 columns: SBD HOTEN TOAN VAN LY HOA SINH SU DIA NGUTN NGUTL NGUTOTAL
        let row = vec![
            s("TP001"),
            s("Nguyễn Thị Lan"),
            Data::Float(8.0),
            Data::Float(7.5),
            Data::Float(9.0),
            Data::Float(6.5),
            Data::Float(5.0),
            Data::Float(4.5),
            Data::Float(8.0),
            Data::Empty,
            Data::Empty,
            Data::Float(7.0), // col 11 → tieng_anh
        ];
        let row = process_separate_scores_row(&row, &p).expect("should parse");
        assert_eq!(row.so_bao_danh, "TP001");
        assert_eq!(row.ho_ten, "Nguyễn Thị Lan");
        assert_eq!(row.ho_ten_ascii, "nguyen thi lan");
        assert_eq!(row.scores.get("toan"), Some(&8.0));
        assert_eq!(row.scores.get("tieng_anh"), Some(&7.0));
        assert!(row.ngay_sinh.is_none());
        assert!(row.ten_cum_thi.is_none());
        assert!(row.gioi_tinh.is_none());
    }

    #[test]
    fn separate_scores_skips_empty_sbd() {
        let p = make_patterns();
        let row = vec![s(""), s("Nguyễn Văn A"), Data::Float(5.0)];
        assert!(process_separate_scores_row(&row, &p).is_none());
    }

    // --- process_mapped_row ---

    #[test]
    fn mapped_row_full_fields() {
        let p = make_patterns();
        let row = vec![
            s("HCM001"),
            s("Trần Thị Bích"),
            s("15/3/1999"),
            s("Cụm thi HCM"),
            s("Nữ"),
            s("Toán: 9.0 Ngữ văn: 8.5 Tiếng Anh: 7.75"),
        ];
        let parsed = process_mapped_row(&row, 0, 1, Some(2), Some(3), Some(4), 5, &p)
            .expect("should parse");
        assert_eq!(parsed.so_bao_danh, "HCM001");
        assert_eq!(parsed.ngay_sinh.as_deref(), Some("15/3/1999"));
        assert_eq!(parsed.ten_cum_thi.as_deref(), Some("Cụm thi HCM"));
        assert_eq!(parsed.gioi_tinh.as_deref(), Some("Nữ"));
        assert_eq!(parsed.scores.get("toan"), Some(&9.0));
        assert_eq!(parsed.scores.get("ngu_van"), Some(&8.5));
        assert_eq!(parsed.scores.get("tieng_anh"), Some(&7.75));
    }

    #[test]
    fn mapped_row_gender_normalisation() {
        let p = make_patterns();
        // Gender "Unknown" → None
        let row = vec![s("ABC"), s("Nguyen Van A"), s(""), s(""), s("Unknown"), s("")];
        let parsed =
            process_mapped_row(&row, 0, 1, Some(2), Some(3), Some(4), 5, &p).expect("should parse");
        assert!(parsed.gioi_tinh.is_none());

        // "Nam" passes through
        let row2 = vec![s("ABC"), s("Nguyen Van A"), s(""), s(""), s("Nam"), s("")];
        let parsed2 =
            process_mapped_row(&row2, 0, 1, Some(2), Some(3), Some(4), 5, &p).expect("should parse");
        assert_eq!(parsed2.gioi_tinh.as_deref(), Some("Nam"));
    }

    #[test]
    fn mapped_row_skips_leaked_header() {
        let p = make_patterns();
        // A leaked header row — SBD cell contains "SOBAODANH"
        let row = vec![s("SOBAODANH"), s("HO_TEN"), s("NGAY_SINH"), s(""), s(""), s("")];
        assert!(process_mapped_row(&row, 0, 1, Some(2), Some(3), Some(4), 5, &p).is_none());
    }

    // --- process_default_row ---

    #[test]
    fn default_row_positional() {
        let p = make_patterns();
        let row = vec![
            s("DN001"),
            s("Lê Văn Long"),
            s("10/5/1998"),
            s("Cụm Đà Nẵng"),
            s("Nam"),
            s("Tiếng Đức: 6.25"),
        ];
        let parsed = process_default_row(&row, &p).expect("should parse");
        assert_eq!(parsed.so_bao_danh, "DN001");
        assert_eq!(parsed.ho_ten_ascii, "le van long");
        assert_eq!(parsed.ngay_sinh.as_deref(), Some("10/5/1998"));
        assert_eq!(parsed.ten_cum_thi.as_deref(), Some("Cụm Đà Nẵng"));
        assert_eq!(parsed.gioi_tinh.as_deref(), Some("Nam"));
        assert_eq!(parsed.scores.get("tieng_duc"), Some(&6.25));
    }

    // --- is_header_row_2016 ---

    #[test]
    fn header_row_detection_2016() {
        assert!(is_header_row_2016(&[s("SOBAODANH"), s("HO_TEN")]));
        assert!(is_header_row_2016(&[s("SBD"), s("HOTEN"), s("TOAN")]));
        assert!(!is_header_row_2016(&[s("12345678"), s("Nguyen Van A")]));
        assert!(!is_header_row_2016(&[s("SBD")])); // too short (< 2 cells)
    }
}
