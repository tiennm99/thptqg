// Package config loads per-dataset parse rules — a port of parser/src/config.rs.
//
// The config deliberately carries no SQL. The table shape, the INSERT and the
// subject regexes are identical for every dataset and live in internal/schema;
// keeping them here meant four copies of the same DDL, which is how the 2016 and
// 2017 schemas drifted apart.
package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SheetMode selects which sheets of a workbook are read.
type SheetMode string

const (
	// SheetModeAll iterates every sheet, which is what recovers the Hanoi and
	// HCM rows that overflow past Excel's 65,536-row cap into a second sheet.
	SheetModeAll SheetMode = "all"
	// SheetModeFirst reads sheet 0 only.
	SheetModeFirst SheetMode = "first"
)

// valid reports whether m is one of the two values the Rust enum accepts.
//
// Checked after decoding rather than via a custom unmarshaler: the decoder
// assigns named string types directly, so a typo would otherwise decode
// silently and be read as "not all" downstream.
func (m SheetMode) valid() bool {
	return m == SheetModeAll || m == SheetModeFirst
}

// DatasetConfig is the per-dataset parse rule set.
type DatasetConfig struct {
	Reader ReaderCfg `yaml:"reader"`
	// Columns holds fixed column indices. Nil when FormatDetection handles
	// per-file mapping — a pointer rather than a value because a zero ColumnMap
	// would silently mean "every column is index 0".
	Columns    *ColumnMap    `yaml:"columns"`
	Validation ValidationCfg `yaml:"validation"`
	Header     HeaderCfg     `yaml:"header"`
	// FormatDetection, when set to "thptqg2016", enables per-file format
	// auto-detection: each file's header row is inspected at runtime to choose
	// the column layout (separate-scores / mapped / default-positional).
	FormatDetection *string `yaml:"format_detection"`
}

type ReaderCfg struct {
	SheetMode SheetMode `yaml:"sheet_mode"`
	// StripBlankRows skips rows where every cell is empty before counting them
	// as source rows (a 2017-old2 quirk).
	StripBlankRows bool `yaml:"strip_blank_rows"`
}

// ColumnMap holds zero-indexed column positions in the source row. Used by the
// 2017 configs; 2016 uses runtime format detection instead.
type ColumnMap struct {
	HoTen     int `yaml:"ho_ten"`
	NgaySinh  int `yaml:"ngay_sinh"`
	SoBaoDanh int `yaml:"so_bao_danh"`
	DiemThi   int `yaml:"diem_thi"`
}

type ValidationCfg struct {
	// RequireNumericSbd mirrors build-database-old.js / -old2.js, which require
	// so_bao_danh to match ^\d+$.
	RequireNumericSbd   bool `yaml:"require_numeric_sbd"`
	RequireNonemptyName bool `yaml:"require_nonempty_name"`
	RequireNonemptySbd  bool `yaml:"require_nonempty_sbd"`
}

type HeaderCfg struct {
	// Tokens are matched against the uppercased first cell to detect a header row.
	Tokens []string `yaml:"tokens"`
}

// Parse decodes a config, rejecting any key the struct does not declare.
//
// Strictness is load-bearing and has its own test. Rust gets it from serde's
// deny_unknown_fields; yaml.v3 needs KnownFields(true) explicitly, and Go YAML
// decoders ignore unknown keys by default. Without it a leftover `schema:`
// mapping would look effective while internal/schema actually drove the build —
// the drift that produced two divergent schemas before the unification.
func Parse(src []byte) (*DatasetConfig, error) {
	var cfg DatasetConfig
	dec := yaml.NewDecoder(bytes.NewReader(src))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if !cfg.Reader.SheetMode.valid() {
		return nil, fmt.Errorf("invalid sheet_mode %q (want %q or %q)",
			cfg.Reader.SheetMode, SheetModeAll, SheetModeFirst)
	}
	return &cfg, nil
}

// Load reads and parses a config file.
func Load(path string) (*DatasetConfig, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}
