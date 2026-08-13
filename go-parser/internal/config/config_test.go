package config

import (
	"os"
	"path/filepath"
	"testing"
)

// sampleYAML mirrors SAMPLE_YAML in parser/src/config.rs.
const sampleYAML = `
reader:
  sheet_mode: all
  strip_blank_rows: false

columns:
  ho_ten: 0
  ngay_sinh: 1
  so_bao_danh: 2
  diem_thi: 3

validation:
  require_numeric_sbd: false
  require_nonempty_name: true
  require_nonempty_sbd: true

header:
  tokens: ["HO_TEN", "HỌ TÊN", "STT"]
`

// TestConfigRoundTrip ports config_round_trip (config.rs:115).
func TestConfigRoundTrip(t *testing.T) {
	cfg, err := Parse([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if cfg.Reader.SheetMode != SheetModeAll {
		t.Errorf("sheet_mode = %q, want all", cfg.Reader.SheetMode)
	}
	if cfg.Reader.StripBlankRows {
		t.Error("strip_blank_rows should be false")
	}
	if cfg.Columns == nil {
		t.Fatal("columns should be present")
	}
	if cfg.Columns.HoTen != 0 {
		t.Errorf("ho_ten = %d, want 0", cfg.Columns.HoTen)
	}
	if cfg.Columns.DiemThi != 3 {
		t.Errorf("diem_thi = %d, want 3", cfg.Columns.DiemThi)
	}
	if cfg.Validation.RequireNumericSbd {
		t.Error("require_numeric_sbd should be false")
	}
	if !cfg.Validation.RequireNonemptyName {
		t.Error("require_nonempty_name should be true")
	}
	if len(cfg.Header.Tokens) != 3 {
		t.Errorf("tokens = %d, want 3", len(cfg.Header.Tokens))
	}
	if cfg.FormatDetection != nil {
		t.Errorf("format_detection = %v, want nil", *cfg.FormatDetection)
	}
}

// TestConfigRejectsLeftoverSQLSections ports config_rejects_leftover_sql_sections
// (config.rs:132). This is the load-bearing one: Rust gets the behaviour free
// from serde's deny_unknown_fields, whereas Go YAML decoders ignore unknown keys
// unless KnownFields(true) is set. Without it a stale schema: mapping would look effective while
// internal/schema silently drove the build — exactly the drift that produced two
// divergent schemas before the unification.
func TestConfigRejectsLeftoverSQLSections(t *testing.T) {
	withDDL := sampleYAML + "\nschema:\n  ddl: \"CREATE TABLE student (so_bao_danh TEXT);\"\n"
	if _, err := Parse([]byte(withDDL)); err == nil {
		t.Fatal("config with a leftover schema: mapping must be rejected")
	}
}

// TestConfigRejectsUnknownScalarKey guards the same property for a stray scalar,
// not just a stray mapping.
func TestConfigRejectsUnknownScalarKey(t *testing.T) {
	withKey := sampleYAML + "\nunexpected_key: 1\n"
	if _, err := Parse([]byte(withKey)); err == nil {
		t.Fatal("config with an unknown scalar key must be rejected")
	}
}

// TestConfigFirstSheetMode ports config_first_sheet_mode (config.rs:140).
func TestConfigFirstSheetMode(t *testing.T) {
	src := []byte(replaceAll(sampleYAML, "sheet_mode: all", "sheet_mode: first"))
	cfg, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if cfg.Reader.SheetMode != SheetModeFirst {
		t.Errorf("sheet_mode = %q, want first", cfg.Reader.SheetMode)
	}
}

// TestConfigRejectsUnknownSheetMode: the Rust enum accepts only "all"/"first",
// so anything else must fail rather than defaulting.
func TestConfigRejectsUnknownSheetMode(t *testing.T) {
	src := []byte(replaceAll(sampleYAML, "sheet_mode: all", "sheet_mode: second"))
	if _, err := Parse(src); err == nil {
		t.Fatal("unknown sheet_mode must be rejected")
	}
}

// TestConfigFormatDetectionField ports config_format_detection_field
// (config.rs:147): a 2016-style config has no columns: mapping at all.
func TestConfigFormatDetectionField(t *testing.T) {
	const src = `
format_detection: thptqg2016

reader:
  sheet_mode: all
  strip_blank_rows: false

validation:
  require_numeric_sbd: false
  require_nonempty_name: true
  require_nonempty_sbd: true

header:
  tokens: ["SBD", "SOBAODANH", "STT"]
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if cfg.FormatDetection == nil || *cfg.FormatDetection != "thptqg2016" {
		t.Errorf("format_detection = %v, want thptqg2016", cfg.FormatDetection)
	}
	if cfg.Columns != nil {
		t.Error("columns must be nil when format_detection drives the layout")
	}
}

// TestLoadRealConfigs loads the shipped configs and asserts the per-dataset
// differences recorded during scouting.
func TestLoadRealConfigs(t *testing.T) {
	root := repoRoot(t)
	want := map[string]struct {
		sheetMode  SheetMode
		stripBlank bool
		numericSbd bool
		hasColumns bool
		formatDet  string
		tokenCount int
	}{
		"2016": {SheetModeAll, false, false, false, "thptqg2016", 6},
		"2017": {SheetModeAll, false, false, true, "", 3},
	}
	for id, w := range want {
		t.Run(id, func(t *testing.T) {
			cfg, err := Load(filepath.Join(root, "go-parser", "configs", id+".yml"))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if cfg.Reader.SheetMode != w.sheetMode {
				t.Errorf("sheet_mode = %q, want %q", cfg.Reader.SheetMode, w.sheetMode)
			}
			if cfg.Reader.StripBlankRows != w.stripBlank {
				t.Errorf("strip_blank_rows = %v, want %v", cfg.Reader.StripBlankRows, w.stripBlank)
			}
			if cfg.Validation.RequireNumericSbd != w.numericSbd {
				t.Errorf("require_numeric_sbd = %v, want %v", cfg.Validation.RequireNumericSbd, w.numericSbd)
			}
			if (cfg.Columns != nil) != w.hasColumns {
				t.Errorf("columns present = %v, want %v", cfg.Columns != nil, w.hasColumns)
			}
			got := ""
			if cfg.FormatDetection != nil {
				got = *cfg.FormatDetection
			}
			if got != w.formatDet {
				t.Errorf("format_detection = %q, want %q", got, w.formatDet)
			}
			if len(cfg.Header.Tokens) != w.tokenCount {
				t.Errorf("tokens = %d, want %d", len(cfg.Header.Tokens), w.tokenCount)
			}
			// Every dataset requires a non-empty name and SBD.
			if !cfg.Validation.RequireNonemptyName || !cfg.Validation.RequireNonemptySbd {
				t.Error("both non-empty validations should be true for every dataset")
			}
		})
	}
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go-parser", "configs")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate repo root")
	return ""
}
