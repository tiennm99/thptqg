// Package schema is the single source of truth for the SQL shape of every
// dataset — a direct port of parser/src/schema.rs.
//
// Every dataset writes into the same 22-column student table. Columns a dataset has no data for bind NULL.
//
// Column provenance:
//
//	ten_cum_thi, gioi_tinh, tieng_duc, tieng_nhat  -> 2016 only
//	khtn, khxh, gdcd, tieng_nga                    -> 2017 datasets only
//	everything else                                -> both
//
// Before this consolidation the DDL, the INSERT and the subject regexes were
// duplicated across four TOML configs, which is how the 2016 and 2017 schemas
// drifted apart. The configs now carry only per-dataset parse rules.
package schema

import "regexp"

// DDL is executed verbatim after the output database is (re)created.
//
// idx_ten_cum_thi is partial, so it holds zero entries on the three 2017
// datasets — where the column is always NULL — while staying useful for the
// 2016 cluster-grouping queries. Partial indexes are SQLite-specific.
//
// Byte-identical to parser/src/schema.rs:26-54; TestDDLMatchesRust enforces it.
const DDL = `
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
`

// IdentityFields are the identity columns, in INSERT parameter order.
var IdentityFields = []string{
	"so_bao_danh",
	"ho_ten",
	"ho_ten_ascii",
	"ngay_sinh",
	"ten_cum_thi",
	"gioi_tinh",
}

// ScoreFields are the subject columns, in INSERT parameter order. Bound NULL
// when a row has no score for that subject.
var ScoreFields = []string{
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
}

// ParamCount is the total bound parameters per row.
const ParamCount = 22

// InsertSQL is a positional INSERT matching IdentityFields then ScoreFields.
//
// OR REPLACE is a behavioural contract, not an optimisation: a repeated SBD
// overwrites the earlier row rather than aborting the transaction, so the last
// file to supply a duplicate wins.
const InsertSQL = `
INSERT OR REPLACE INTO student
  (so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh, ten_cum_thi, gioi_tinh,
   toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn,
   lich_su, dia_ly, gdcd, khxh,
   tieng_anh, tieng_phap, tieng_nga, tieng_duc, tieng_nhat, tieng_trung)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

// scorePatternSources holds the regex per subject, applied to the DIEM_THI cell
// text. Copied verbatim from parser/src/schema.rs:126-143 — the literals contain
// Vietnamese text and must never be retyped.
//
// Every pattern runs against every dataset. A subject absent from a given exam
// year simply never matches and stays NULL: 2016 files contain no "KHTN:" or
// "Tiếng Nga:" tokens, and 2017 files contain no "Tiếng Đức:" or "Tiếng Nhật:".
//
// Go's regexp and Rust's regex crate are both RE2, and these patterns use no
// backreferences, lookaround or Unicode classes, so they port with zero risk.
var scorePatternSources = map[string]string{
	"toan":        `Toán:\s*(\d+(?:\.\d+)?)`,
	"ngu_van":     `Ngữ văn:\s*(\d+(?:\.\d+)?)`,
	"vat_ly":      `Vật lí:\s*(\d+(?:\.\d+)?)`,
	"hoa_hoc":     `Hóa học:\s*(\d+(?:\.\d+)?)`,
	"sinh_hoc":    `Sinh học:\s*(\d+(?:\.\d+)?)`,
	"khtn":        `KHTN:\s*(\d+(?:\.\d+)?)`,
	"lich_su":     `Lịch sử:\s*(\d+(?:\.\d+)?)`,
	"dia_ly":      `Địa lí:\s*(\d+(?:\.\d+)?)`,
	"gdcd":        `GDCD:\s*(\d+(?:\.\d+)?)`,
	"khxh":        `KHXH:\s*(\d+(?:\.\d+)?)`,
	"tieng_anh":   `Tiếng Anh:\s*(\d+(?:\.\d+)?)`,
	"tieng_phap":  `Tiếng Pháp:\s*(\d+(?:\.\d+)?)`,
	"tieng_nga":   `Tiếng Nga:\s*(\d+(?:\.\d+)?)`,
	"tieng_duc":   `Tiếng Đức:\s*(\d+(?:\.\d+)?)`,
	"tieng_nhat":  `Tiếng Nhật:\s*(\d+(?:\.\d+)?)`,
	"tieng_trung": `Tiếng Trung:\s*(\d+(?:\.\d+)?)`,
}

// ScorePatterns holds the compiled subject regexes, compiled once at init.
// Rust compiles them once per run in CompiledPatterns::new; a package-level map
// is the equivalent for a single-threaded CLI.
var ScorePatterns = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(scorePatternSources))
	for field, src := range scorePatternSources {
		out[field] = regexp.MustCompile(src)
	}
	return out
}()
