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
// One table and no secondary indexes. The browser downloads this file and
// queries it in memory, so an index buys a scan that already takes a few
// hundred milliseconds while costing tens of megabytes that every visitor
// pays for on the network.
//
// That is a reversal. An earlier design read the file over HTTP range
// requests, where any unindexed query pulls the whole table down, and it
// carried a name_word table — one row per word of every name, ~3.5 million of
// them — plus partial indexes on the score columns. Those made range-request
// queries seek instead of scan, and together they were more than half the
// published file.
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
