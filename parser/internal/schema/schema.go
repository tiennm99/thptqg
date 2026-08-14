// Package schema is the single source of truth for the SQL shape of every
// dataset.
//
// Every dataset writes into the same 22-column student table. Columns a dataset has no data for bind NULL.
//
// Column provenance:
//
//	ten_cum_thi, gioi_tinh  -> 2016 only
//	khtn, khxh, gdcd        -> 2017 only
//	everything else         -> both, all six languages included
//
// The DDL, the INSERT and the subject regexes belong here and nowhere else. One
// copy per dataset is what let the 2016 and 2017 schemas drift apart; the
// per-dataset configs carry only parse rules.
package schema

import "regexp"

// DDL is executed verbatim after the output database is (re)created.
//
// Every index here is chosen for a database read over HTTP range requests,
// where an unindexed query downloads the table. The rules that follow from
// that:
//
//   - No index on ho_ten or ho_ten_ascii. Neither substring nor prefix LIKE can
//     use one (SQLite's LIKE optimisation needs a NOCASE index or
//     case_sensitive_like), so both scanned the whole table. name_word replaces
//     them.
//   - idx_ten_cum_thi is partial, so it holds zero entries on the 2017 dataset
//     — where the column is always NULL — while serving the 2016 cluster
//     grouping. Partial indexes are SQLite-specific.
//
// This text is frozen: it decides the shape of every database the parser
// produces. TestDDLIsFrozen holds an independent copy so any edit has to be
// deliberate.
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
CREATE INDEX idx_ten_cum_thi  ON student(ten_cum_thi) WHERE ten_cum_thi IS NOT NULL;

CREATE TABLE name_word (
  word          TEXT NOT NULL,
  so_bao_danh   TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,
  PRIMARY KEY (word, so_bao_danh)
) WITHOUT ROWID;

CREATE TABLE name_word_freq (
  word TEXT PRIMARY KEY,
  n    INTEGER NOT NULL
) WITHOUT ROWID;
`

// PostLoadSQL runs once the student rows are in, before VACUUM.
//
// The frequency table is what lets the site pick which word of a query to seek
// on: the vocabulary is about 4,400 words and the rarest word of a real query
// matches a few hundred rows, so seeking on it and filtering the rest inside
// name_word keeps a search to a few hundred kilobytes.
//
// The three score indexes are partial for the same reason idx_ten_cum_thi is:
// each covers only the exam year that has the column, and each costs about
// 4 MB. They exist so the SQL presets that rank by these columns seek instead
// of scanning 127 MB.
const PostLoadSQL = `
INSERT INTO name_word_freq (word, n)
  SELECT word, COUNT(*) FROM name_word GROUP BY word;
CREATE INDEX idx_toan ON student(toan) WHERE toan IS NOT NULL;
CREATE INDEX idx_khtn ON student(khtn) WHERE khtn IS NOT NULL;
CREATE INDEX idx_khxh ON student(khxh) WHERE khxh IS NOT NULL;
`

// NameWordInsertSQL adds one row per distinct word of a candidate's ASCII name.
//
// ho_ten_ascii is carried along deliberately: a query with several words seeks
// on the rarest one and filters the others against this copy, so the whole
// match happens inside one b-tree and only the surviving rows are read from
// student.
const NameWordInsertSQL = `
INSERT OR IGNORE INTO name_word (word, so_bao_danh, ho_ten_ascii) VALUES (?, ?, ?)
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
// text. The literals contain Vietnamese subject names exactly as they appear in
// the source files — copy them, never retype them.
//
// Every pattern runs against every dataset. A subject absent from a given exam
// year simply never matches and stays NULL: 2016 files contain no "KHTN:",
// "KHXH:" or "GDCD:" tokens, since those combined papers did not exist yet.
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

// ScorePatterns holds the subject regexes, compiled once at package init.
var ScorePatterns = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(scorePatternSources))
	for field, src := range scorePatternSources {
		out[field] = regexp.MustCompile(src)
	}
	return out
}()
