use std::collections::HashMap;
use std::fs;
use std::path::Path;

use serde::Deserialize;

use crate::error::BuildError;

// ---------------------------------------------------------------------------
// Top-level dataset configuration loaded from a .toml file
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize, Clone)]
pub struct DatasetConfig {
    pub reader: ReaderCfg,
    /// Fixed column indices. Optional when format_detection handles per-file mapping.
    #[serde(default)]
    pub columns: Option<ColumnMap>,
    pub validation: ValidationCfg,
    pub header: HeaderCfg,
    pub schema: SchemaCfg,
    /// field name → regex source string (one entry per scoreable subject)
    pub scores: HashMap<String, String>,
    pub insert: InsertCfg,
    /// When set to "thptqg2016", enables per-file format auto-detection.
    /// Each file's header row is inspected at runtime to choose the right
    /// column layout (separate-scores / mapped / default-positional).
    #[serde(default)]
    pub format_detection: Option<String>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ReaderCfg {
    /// "all" → iterate every sheet (handles HCM/HN overflow); "first" → sheet 0 only
    pub sheet_mode: SheetMode,
    /// If true, skip rows where every cell is empty/null before counting (data-old2 quirk)
    pub strip_blank_rows: bool,
}

#[derive(Debug, Deserialize, Clone, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum SheetMode {
    All,
    First,
}

/// Zero-indexed column positions in the source spreadsheet row.
/// Used by thptqg2017 configs. thptqg2016 uses runtime format detection instead.
#[derive(Debug, Deserialize, Clone)]
pub struct ColumnMap {
    pub ho_ten: usize,
    pub ngay_sinh: usize,
    pub so_bao_danh: usize,
    pub diem_thi: usize,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ValidationCfg {
    /// build-database-old.js / -old2.js require soBaoDanh to match ^\d+$
    pub require_numeric_sbd: bool,
    pub require_nonempty_name: bool,
    pub require_nonempty_sbd: bool,
}

#[derive(Debug, Deserialize, Clone)]
pub struct HeaderCfg {
    /// Tokens to match against row[0].to_uppercase() to detect a header row
    pub tokens: Vec<String>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct SchemaCfg {
    /// DDL executed verbatim before inserts (CREATE TABLE + CREATE INDEX)
    pub ddl: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct InsertCfg {
    /// Parameterised INSERT OR REPLACE SQL using :named_param style
    pub sql: String,
}

// ---------------------------------------------------------------------------
// Loader
// ---------------------------------------------------------------------------

pub fn load_config(path: &Path) -> Result<DatasetConfig, BuildError> {
    let text = fs::read_to_string(path).map_err(|e| BuildError::Io {
        path: path.display().to_string(),
        source: e,
    })?;
    let cfg: DatasetConfig = toml::from_str(&text)?;
    Ok(cfg)
}

// ---------------------------------------------------------------------------
// Unit tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    const SAMPLE_TOML: &str = r#"
[reader]
sheet_mode = "all"
strip_blank_rows = false

[columns]
ho_ten      = 0
ngay_sinh   = 1
so_bao_danh = 2
diem_thi    = 3

[validation]
require_numeric_sbd   = false
require_nonempty_name = true
require_nonempty_sbd  = true

[header]
tokens = ["HO_TEN", "HỌ TÊN", "STT"]

[schema]
ddl = "CREATE TABLE student (so_bao_danh TEXT PRIMARY KEY);"

[scores]
toan    = 'Toán:\s*(\d+(?:\.\d+)?)'
ngu_van = 'Ngữ văn:\s*(\d+(?:\.\d+)?)'

[insert]
sql = "INSERT OR REPLACE INTO student (so_bao_danh) VALUES (:so_bao_danh)"
"#;

    #[test]
    fn config_round_trip() {
        let cfg: DatasetConfig = toml::from_str(SAMPLE_TOML).expect("parse failed");
        assert_eq!(cfg.reader.sheet_mode, SheetMode::All);
        assert!(!cfg.reader.strip_blank_rows);
        let cols = cfg.columns.as_ref().unwrap();
        assert_eq!(cols.ho_ten, 0);
        assert_eq!(cols.diem_thi, 3);
        assert!(!cfg.validation.require_numeric_sbd);
        assert!(cfg.validation.require_nonempty_name);
        assert_eq!(cfg.header.tokens.len(), 3);
        assert!(cfg.scores.contains_key("toan"));
        assert!(cfg.scores.contains_key("ngu_van"));
        assert!(cfg.format_detection.is_none());
    }

    #[test]
    fn config_first_sheet_mode() {
        let toml_str = SAMPLE_TOML.replace(r#"sheet_mode = "all""#, r#"sheet_mode = "first""#);
        let cfg: DatasetConfig = toml::from_str(&toml_str).expect("parse failed");
        assert_eq!(cfg.reader.sheet_mode, SheetMode::First);
    }

    #[test]
    fn config_format_detection_field() {
        // Configs without [columns] and with format_detection = "thptqg2016" parse correctly
        let toml_str = r#"
format_detection = "thptqg2016"

[reader]
sheet_mode = "all"
strip_blank_rows = false

[validation]
require_numeric_sbd   = false
require_nonempty_name = true
require_nonempty_sbd  = true

[header]
tokens = ["SBD", "SOBAODANH", "STT"]

[schema]
ddl = "CREATE TABLE student (so_bao_danh TEXT PRIMARY KEY);"

[scores]
toan = 'Toán:\s*(\d+(?:\.\d+)?)'

[insert]
sql = "INSERT OR REPLACE INTO student (so_bao_danh) VALUES (?)"
"#;
        let cfg: DatasetConfig = toml::from_str(toml_str).expect("parse failed");
        assert_eq!(cfg.format_detection.as_deref(), Some("thptqg2016"));
        assert!(cfg.columns.is_none());
    }
}
