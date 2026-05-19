/// Row transformation: ascii normalisation, score regex parsing, validation.
///
/// `to_ascii` replicates build-lib.js `toAscii` exactly:
///   str.normalize("NFD").replace(/[̀-ͯ]/g,"").replace(/đ/gi,"d").toLowerCase()
use std::collections::HashMap;

use regex::Regex;
use unicode_normalization::UnicodeNormalization;

use crate::config::{DatasetConfig, ValidationCfg};
use crate::error::BuildError;

// ---------------------------------------------------------------------------
// Compiled score patterns (built once at startup from config)
// ---------------------------------------------------------------------------

pub struct CompiledPatterns {
    /// Ordered list so INSERT column order is deterministic
    pub patterns: Vec<(String, Regex)>,
}

impl CompiledPatterns {
    pub fn new(scores: &HashMap<String, String>) -> Result<Self, BuildError> {
        let mut patterns = Vec::with_capacity(scores.len());
        for (field, src) in scores {
            let re = Regex::new(src).map_err(|e| BuildError::Regex {
                pattern: src.clone(),
                source: e,
            })?;
            patterns.push((field.clone(), re));
        }
        // Sort for deterministic order across HashMap iteration
        patterns.sort_by(|a, b| a.0.cmp(&b.0));
        Ok(Self { patterns })
    }
}

// ---------------------------------------------------------------------------
// to_ascii — must be byte-for-byte equivalent to build-lib.js toAscii
// ---------------------------------------------------------------------------

/// Normalise a Vietnamese name to an ASCII slug.
///
/// Algorithm mirrors the JavaScript `toAscii` in build-lib.js:
///   1. NFD decompose (splits base + combining diacritics)
///   2. Drop all Unicode combining marks (U+0300–U+036F)
///   3. Replace đ/Đ with d (NFD does not decompose đ)
///   4. Lowercase
pub fn to_ascii(s: &str) -> String {
    // Step 1 + 2: NFD then filter out combining marks (Unicode category M)
    let decomposed: String = s
        .nfd()
        .filter(|c| !('\u{0300}'..='\u{036f}').contains(c))
        .collect();

    // Step 3: đ/Đ are not decomposed by NFD — replace explicitly
    let replaced = decomposed.replace(['đ', 'Đ'], "d");

    // Step 4: lowercase
    replaced.to_lowercase()
}

// ---------------------------------------------------------------------------
// Parsed row ready for DB insert
// ---------------------------------------------------------------------------

pub struct ParsedRow {
    pub so_bao_danh: String,
    pub ho_ten: String,
    pub ho_ten_ascii: String,
    pub ngay_sinh: Option<String>,
    /// Subject field → float value; absent subjects not in map → NULL
    pub scores: HashMap<String, f64>,
}

// ---------------------------------------------------------------------------
// Row validation — mirrors the per-script skip logic
// ---------------------------------------------------------------------------

/// Returns `None` when the row should be skipped entirely (before sourceRows counter).
/// Returns `Some(reason)` when the row should be counted as sourceRows but skipped.
#[derive(Debug, PartialEq, Eq)]
pub enum SkipReason {
    /// Row is fully blank (data-old2 only, before sourceRows counter)
    BlankRow,
    /// soBaoDanh or hoTen empty/missing
    EmptyField,
    /// soBaoDanh contains non-digit characters (data-old / data-old2 guard)
    NonNumericSbd,
}

/// Validates a raw cell slice against the dataset's `ValidationCfg`.
/// Returns `Ok(())` on pass, `Err(SkipReason)` on fail.
pub fn validate_row(
    ho_ten: &str,
    so_bao_danh: &str,
    cfg: &ValidationCfg,
    strip_blank_rows: bool,
    all_blank: bool,
) -> Result<(), SkipReason> {
    // data-old2: skip fully blank rows BEFORE counting sourceRows
    if strip_blank_rows && all_blank {
        return Err(SkipReason::BlankRow);
    }

    if cfg.require_nonempty_sbd && so_bao_danh.is_empty() {
        return Err(SkipReason::EmptyField);
    }
    if cfg.require_nonempty_name && ho_ten.is_empty() {
        return Err(SkipReason::EmptyField);
    }
    if cfg.require_numeric_sbd && !so_bao_danh.chars().all(|c| c.is_ascii_digit()) {
        return Err(SkipReason::NonNumericSbd);
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// Score parsing — mirrors build-lib.js parseScores
// ---------------------------------------------------------------------------

/// Parse a DIEM_THI cell string and extract matching subject scores.
pub fn parse_scores(diem_thi: &str, patterns: &CompiledPatterns) -> HashMap<String, f64> {
    let mut out = HashMap::new();
    for (field, re) in &patterns.patterns {
        if let Some(caps) = re.captures(diem_thi) {
            if let Some(m) = caps.get(1) {
                if let Ok(v) = m.as_str().parse::<f64>() {
                    if v.is_finite() {
                        out.insert(field.clone(), v);
                    }
                }
            }
        }
    }
    out
}

// ---------------------------------------------------------------------------
// Full row transform
// ---------------------------------------------------------------------------

/// Extract and transform one spreadsheet row into a `ParsedRow`.
/// `raw` is the full cell slice; column indices come from `cfg.columns`.
pub fn transform_row(
    raw: &[calamine::Data],
    cfg: &DatasetConfig,
    patterns: &CompiledPatterns,
) -> ParsedRow {
    let get = |idx: usize| -> String {
        raw.get(idx)
            .map(|cell| cell.to_string().trim().to_owned())
            .unwrap_or_default()
    };

    let ho_ten = get(cfg.columns.ho_ten);
    let ngay_sinh = get(cfg.columns.ngay_sinh);
    let so_bao_danh = get(cfg.columns.so_bao_danh);
    let diem_thi = raw
        .get(cfg.columns.diem_thi)
        .map(|c| c.to_string())
        .unwrap_or_default();

    let ho_ten_ascii = to_ascii(&ho_ten);
    let scores = parse_scores(&diem_thi, patterns);
    let ngay_sinh_opt = if ngay_sinh.is_empty() {
        None
    } else {
        Some(ngay_sinh)
    };

    ParsedRow {
        so_bao_danh,
        ho_ten,
        ho_ten_ascii,
        ngay_sinh: ngay_sinh_opt,
        scores,
    }
}

// ---------------------------------------------------------------------------
// Unit tests — 20 cases for to_ascii (real Vietnamese names)
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    // Helper: assert to_ascii(input) == expected
    fn check(input: &str, expected: &str) {
        assert_eq!(
            to_ascii(input),
            expected,
            "to_ascii({input:?}) expected {expected:?}"
        );
    }

    #[test]
    fn ascii_plain_latin() {
        check("Nguyen Van A", "nguyen van a");
    }

    #[test]
    fn ascii_nguyen_thi_hoa() {
        check("Nguyễn Thị Hoa", "nguyen thi hoa");
    }

    #[test]
    fn ascii_tran_van_duc() {
        // đ/Đ replacement
        check("Trần Văn Đức", "tran van duc");
    }

    #[test]
    fn ascii_le_thi_my_duyen() {
        check("Lê Thị Mỹ Duyên", "le thi my duyen");
    }

    #[test]
    fn ascii_pham_thi_lan() {
        check("Phạm Thị Lan", "pham thi lan");
    }

    #[test]
    fn ascii_bui_thi_thu() {
        check("Bùi Thị Thu", "bui thi thu");
    }

    #[test]
    fn ascii_hoang_van_truong() {
        check("Hoàng Văn Trường", "hoang van truong");
    }

    #[test]
    fn ascii_do_thi_ngan() {
        // Đ uppercase at start
        check("Đỗ Thị Ngân", "do thi ngan");
    }

    #[test]
    fn ascii_nguyen_van_khanh() {
        check("Nguyễn Văn Khánh", "nguyen van khanh");
    }

    #[test]
    fn ascii_trinh_thi_bich_ngoc() {
        check("Trịnh Thị Bích Ngọc", "trinh thi bich ngoc");
    }

    #[test]
    fn ascii_vu_thi_dieu() {
        // ề = e + combining grave + combining circumflex (after NFD)
        check("Vũ Thị Diệu", "vu thi dieu");
    }

    #[test]
    fn ascii_nguyen_thi_tuong_vi() {
        check("Nguyễn Thị Tường Vi", "nguyen thi tuong vi");
    }

    #[test]
    fn ascii_lowercase_d_stroke() {
        // Lowercase đ → d
        check("đặng thị hằng", "dang thi hang");
    }

    #[test]
    fn ascii_uppercase_d_stroke() {
        check("ĐẶNG THỊ HẰNG", "dang thi hang");
    }

    #[test]
    fn ascii_mixed_case() {
        check("NGUYỄN VĂN AN", "nguyen van an");
    }

    #[test]
    fn ascii_tran_thi_kim_anh() {
        check("Trần Thị Kim Anh", "tran thi kim anh");
    }

    #[test]
    fn ascii_nguyen_thi_phuong_thao() {
        check("Nguyễn Thị Phương Thảo", "nguyen thi phuong thao");
    }

    #[test]
    fn ascii_le_van_long() {
        check("Lê Văn Long", "le van long");
    }

    #[test]
    fn ascii_vo_thi_xuan_mai() {
        check("Võ Thị Xuân Mai", "vo thi xuan mai");
    }

    #[test]
    fn ascii_empty_string() {
        check("", "");
    }

    // --- Score parsing tests ---

    fn make_patterns() -> CompiledPatterns {
        let mut map = HashMap::new();
        map.insert("toan".into(), r"Toán:\s*(\d+(?:\.\d+)?)".into());
        map.insert("ngu_van".into(), r"Ngữ văn:\s*(\d+(?:\.\d+)?)".into());
        map.insert("vat_ly".into(), r"Vật lí:\s*(\d+(?:\.\d+)?)".into());
        CompiledPatterns::new(&map).unwrap()
    }

    #[test]
    fn parse_scores_single() {
        let p = make_patterns();
        let s = "Toán: 8.5";
        let scores = parse_scores(s, &p);
        assert_eq!(scores.get("toan"), Some(&8.5));
        assert!(scores.get("ngu_van").is_none());
    }

    #[test]
    fn parse_scores_multiple() {
        let p = make_patterns();
        let s = "Toán: 7.25  Ngữ văn: 6.0  Vật lí: 9";
        let scores = parse_scores(s, &p);
        assert_eq!(scores.get("toan"), Some(&7.25));
        assert_eq!(scores.get("ngu_van"), Some(&6.0));
        assert_eq!(scores.get("vat_ly"), Some(&9.0));
    }

    #[test]
    fn parse_scores_empty_cell() {
        let p = make_patterns();
        let scores = parse_scores("", &p);
        assert!(scores.is_empty());
    }

    // --- Validation tests ---

    fn default_validation() -> ValidationCfg {
        ValidationCfg {
            require_numeric_sbd: false,
            require_nonempty_name: true,
            require_nonempty_sbd: true,
        }
    }

    #[test]
    fn validate_ok() {
        let v = default_validation();
        assert!(validate_row("Nguyen Van A", "12345678", &v, false, false).is_ok());
    }

    #[test]
    fn validate_empty_sbd() {
        let v = default_validation();
        assert_eq!(
            validate_row("Nguyen Van A", "", &v, false, false),
            Err(SkipReason::EmptyField)
        );
    }

    #[test]
    fn validate_empty_name() {
        let v = default_validation();
        assert_eq!(
            validate_row("", "12345678", &v, false, false),
            Err(SkipReason::EmptyField)
        );
    }

    #[test]
    fn validate_non_numeric_sbd_rejected() {
        let mut v = default_validation();
        v.require_numeric_sbd = true;
        assert_eq!(
            validate_row("Nguyen Van A", "12AB5678", &v, false, false),
            Err(SkipReason::NonNumericSbd)
        );
    }

    #[test]
    fn validate_numeric_sbd_accepted() {
        let mut v = default_validation();
        v.require_numeric_sbd = true;
        assert!(validate_row("Nguyen Van A", "12345678", &v, false, false).is_ok());
    }

    #[test]
    fn validate_blank_row_skipped() {
        let v = default_validation();
        // strip_blank_rows=true AND all_blank=true → BlankRow
        assert_eq!(
            validate_row("", "", &v, true, true),
            Err(SkipReason::BlankRow)
        );
    }
}
