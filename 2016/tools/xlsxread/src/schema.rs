//! Canonical `student` table definition — the single source of truth for the
//! SQL shape of every dataset.
//!
//! Every dataset (2016, 2017, 2017-old, 2017-old2) is written into this same
//! 22-column table. Columns a dataset has no data for bind NULL, which costs
//! ~1 byte per row in SQLite's record header.
//!
//! Column provenance:
//!   ten_cum_thi, gioi_tinh, tieng_duc, tieng_nhat  → 2016 only
//!   khtn, khxh, gdcd, tieng_nga                    → 2017 datasets only
//!   everything else                                → both
//!
//! Before this module existed the DDL, the INSERT statement and the subject
//! regex table were duplicated across four TOML configs, which is how the 2016
//! and 2017 schemas drifted apart in the first place. The configs now carry only
//! per-dataset parse rules.
// ---------------------------------------------------------------------------
// DDL
// ---------------------------------------------------------------------------

/// Executed verbatim after the output database is (re)created.
///
/// `idx_ten_cum_thi` is partial so it holds zero entries on the three 2017
/// datasets — where the column is always NULL — while staying fully useful for
/// the 2016 cluster-grouping queries.
pub const DDL: &str = "
CREATE TABLE student (
  so_bao_danh   TEXT PRIMARY KEY,
  ho_ten        TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,
  ngay_sinh     TEXT,
  ten_cum_thi   TEXT,
  gioi_tinh     TEXT,
  toan          REAL,
  ngu_van       REAL,
  vat_ly        REAL,
  hoa_hoc       REAL,
  sinh_hoc      REAL,
  khtn          REAL,
  lich_su       REAL,
  dia_ly        REAL,
  gdcd          REAL,
  khxh          REAL,
  tieng_anh     REAL,
  tieng_phap    REAL,
  tieng_nga     REAL,
  tieng_duc     REAL,
  tieng_nhat    REAL,
  tieng_trung   REAL
);
CREATE INDEX idx_ho_ten       ON student(ho_ten);
CREATE INDEX idx_ho_ten_ascii ON student(ho_ten_ascii);
CREATE INDEX idx_ten_cum_thi  ON student(ten_cum_thi) WHERE ten_cum_thi IS NOT NULL;
";

// ---------------------------------------------------------------------------
// Column order
// ---------------------------------------------------------------------------

/// Identity columns, in INSERT parameter order.
pub const IDENTITY_FIELDS: &[&str] = &[
    "so_bao_danh",
    "ho_ten",
    "ho_ten_ascii",
    "ngay_sinh",
    "ten_cum_thi",
    "gioi_tinh",
];

/// Subject columns, in INSERT parameter order. Bound as NULL when a row has no
/// score for that subject.
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
    "tieng_duc",
    "tieng_nhat",
    "tieng_trung",
];

/// Total bound parameters per row.
pub const PARAM_COUNT: usize = IDENTITY_FIELDS.len() + SCORE_FIELDS.len();

// ---------------------------------------------------------------------------
// INSERT
// ---------------------------------------------------------------------------

/// Positional INSERT matching IDENTITY_FIELDS followed by SCORE_FIELDS.
///
/// `OR REPLACE` preserves the pre-existing behaviour where a repeated SBD
/// overwrites the earlier row rather than aborting the transaction.
pub const INSERT_SQL: &str = "
INSERT OR REPLACE INTO student
  (so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh, ten_cum_thi, gioi_tinh,
   toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn,
   lich_su, dia_ly, gdcd, khxh,
   tieng_anh, tieng_phap, tieng_nga, tieng_duc, tieng_nhat, tieng_trung)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
";

// ---------------------------------------------------------------------------
// Subject score patterns
// ---------------------------------------------------------------------------

/// Regex per subject, applied to the DIEM_THI cell text.
///
/// Every pattern runs against every dataset. A subject absent from a given exam
/// year simply never matches and stays NULL — 2016 source files contain no
/// "KHTN:" or "Tiếng Nga:" tokens, and 2017 files contain no "Tiếng Đức:" or
/// "Tiếng Nhật:". The parity check asserts those counts are exactly zero rather
/// than assuming it.
///
/// Order here is irrelevant (matching is by name); SCORE_FIELDS fixes the
/// INSERT order.
pub const SCORE_PATTERNS: &[(&str, &str)] = &[
    ("toan", r"Toán:\s*(\d+(?:\.\d+)?)"),
    ("ngu_van", r"Ngữ văn:\s*(\d+(?:\.\d+)?)"),
    ("vat_ly", r"Vật lí:\s*(\d+(?:\.\d+)?)"),
    ("hoa_hoc", r"Hóa học:\s*(\d+(?:\.\d+)?)"),
    ("sinh_hoc", r"Sinh học:\s*(\d+(?:\.\d+)?)"),
    ("khtn", r"KHTN:\s*(\d+(?:\.\d+)?)"),
    ("lich_su", r"Lịch sử:\s*(\d+(?:\.\d+)?)"),
    ("dia_ly", r"Địa lí:\s*(\d+(?:\.\d+)?)"),
    ("gdcd", r"GDCD:\s*(\d+(?:\.\d+)?)"),
    ("khxh", r"KHXH:\s*(\d+(?:\.\d+)?)"),
    ("tieng_anh", r"Tiếng Anh:\s*(\d+(?:\.\d+)?)"),
    ("tieng_phap", r"Tiếng Pháp:\s*(\d+(?:\.\d+)?)"),
    ("tieng_nga", r"Tiếng Nga:\s*(\d+(?:\.\d+)?)"),
    ("tieng_duc", r"Tiếng Đức:\s*(\d+(?:\.\d+)?)"),
    ("tieng_nhat", r"Tiếng Nhật:\s*(\d+(?:\.\d+)?)"),
    ("tieng_trung", r"Tiếng Trung:\s*(\d+(?:\.\d+)?)"),
];

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    /// The INSERT placeholder count, the named column list and the field
    /// constants must agree, or rows silently land in the wrong columns.
    #[test]
    fn insert_matches_field_order() {
        assert_eq!(PARAM_COUNT, 22);
        assert_eq!(INSERT_SQL.matches('?').count(), PARAM_COUNT);

        let named = INSERT_SQL
            .split_once('(')
            .and_then(|(_, rest)| rest.split_once(')'))
            .map(|(cols, _)| cols)
            .expect("INSERT must contain a column list");

        let listed: Vec<&str> = named
            .split(',')
            .map(str::trim)
            .filter(|s| !s.is_empty())
            .collect();

        let expected: Vec<&str> = IDENTITY_FIELDS
            .iter()
            .chain(SCORE_FIELDS.iter())
            .copied()
            .collect();

        assert_eq!(listed, expected);
    }

    /// Every subject column must have a pattern and vice versa.
    #[test]
    fn score_patterns_cover_score_fields() {
        assert_eq!(SCORE_PATTERNS.len(), SCORE_FIELDS.len());
        for (field, _) in SCORE_PATTERNS {
            assert!(
                SCORE_FIELDS.contains(field),
                "pattern {field} has no column"
            );
        }
        for field in SCORE_FIELDS {
            assert!(
                SCORE_PATTERNS.iter().any(|(f, _)| f == field),
                "column {field} has no pattern"
            );
        }
    }

    /// Every column named in the DDL must be bound by the INSERT.
    #[test]
    fn ddl_columns_match_insert() {
        for field in IDENTITY_FIELDS.iter().chain(SCORE_FIELDS.iter()) {
            assert!(DDL.contains(field), "DDL missing column {field}");
        }
    }

    #[test]
    fn score_patterns_compile() {
        for (field, src) in SCORE_PATTERNS {
            regex::Regex::new(src).unwrap_or_else(|e| panic!("{field}: {e}"));
        }
    }
}
