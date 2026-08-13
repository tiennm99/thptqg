// Package transform performs row transformation: ASCII normalisation, score
// regex parsing, and validation — a port of parser/src/transform.rs.
//
// ToAscii replicates build-lib.js toAscii exactly:
//
//	str.normalize("NFD").replace(/[̀-ͯ]/g,"").replace(/đ/gi,"d").toLowerCase()
package transform

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/tiennm99/thptqg/go-parser/internal/config"
	"github.com/tiennm99/thptqg/go-parser/internal/reader"
	"github.com/tiennm99/thptqg/go-parser/internal/schema"
)

// ToAscii normalises a Vietnamese name to an ASCII slug.
//
//  1. NFD decompose (splits base + combining diacritics)
//  2. Drop combining marks in U+0300..U+036F
//  3. Replace đ/Đ with d (NFD does not decompose them)
//  4. Lowercase
//
// Step 2 filters a LITERAL CODEPOINT RANGE, not a Unicode category. The inline
// comment at transform.rs:53 says "Unicode category M", but the code at :56 is
// the specification and it checks '\u{0300}'..='\u{036f}'. unicode.Is(unicode.Mn, r)
// is strictly broader and would strip marks Rust keeps, silently changing
// ho_ten_ascii — the column the site's accent-insensitive search runs on.
//
// Step order matters and mirrors transform.rs:54-63: the đ/Đ replacement happens
// before lowercasing.
func ToAscii(s string) string {
	if s == "" {
		return ""
	}
	decomposed := norm.NFD.String(s)

	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if r >= 0x0300 && r <= 0x036F {
			continue // combining mark, in the range Rust drops
		}
		switch r {
		case 'đ', 'Đ':
			b.WriteByte('d')
		default:
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

// ParsedRow is one row ready for insertion.
type ParsedRow struct {
	SoBaoDanh  string
	HoTen      string
	HoTenAscii string
	NgaySinh   *string
	// TenCumThi is 2016 only: examination cluster name (TEN_CUMTHI column).
	TenCumThi *string
	// GioiTinh is 2016 only: gender, normalised to "Nam"/"Nữ" or nil.
	GioiTinh *string
	// Scores maps subject field -> value. Absent subjects are simply missing and
	// bind NULL.
	Scores map[string]float64
}

// SkipReason says why a row was skipped, or SkipNone when it passed.
//
// The distinction is load-bearing for the printed counters, and the two
// non-blank reasons are counted as source rows while BlankRow is not — but note
// that split lives in the CALLER, not here. parser/src/main.rs has two call
// sites with opposite outcomes for BlankRow: at :135-137 it returns before the
// counter at :140, while at :151 it matches Err(BlankRow) => {} and falls
// through to transform and insert. The build loop must reproduce both.
type SkipReason int

const (
	SkipNone SkipReason = iota
	// SkipBlankRow: row is fully blank (2017-old2 only, checked before the
	// source-row counter).
	SkipBlankRow
	// SkipEmptyField: so_bao_danh or ho_ten empty/missing.
	SkipEmptyField
	// SkipNonNumericSbd: so_bao_danh contains non-digit characters
	// (2017-old / 2017-old2 guard).
	SkipNonNumericSbd
)

func (s SkipReason) String() string {
	switch s {
	case SkipNone:
		return "none"
	case SkipBlankRow:
		return "blank_row"
	case SkipEmptyField:
		return "empty_field"
	case SkipNonNumericSbd:
		return "non_numeric_sbd"
	}
	return "unknown"
}

// ValidateRow checks a row against the dataset's validation rules.
//
// Signature mirrors transform.rs:101-107, taking stripBlankRows and allBlank
// explicitly; a shorter signature could not express both blank-row paths.
func ValidateRow(hoTen, soBaoDanh string, cfg *config.ValidationCfg, stripBlankRows, allBlank bool) SkipReason {
	// 2017-old2: skip fully blank rows BEFORE counting source rows.
	if stripBlankRows && allBlank {
		return SkipBlankRow
	}
	if cfg.RequireNonemptySbd && soBaoDanh == "" {
		return SkipEmptyField
	}
	if cfg.RequireNonemptyName && hoTen == "" {
		return SkipEmptyField
	}
	if cfg.RequireNumericSbd && !allASCIIDigits(soBaoDanh) {
		return SkipNonNumericSbd
	}
	return SkipNone
}

// allASCIIDigits mirrors Rust's chars().all(|c| c.is_ascii_digit()).
//
// Deliberately not strconv.Atoi: Atoi accepts a leading sign, so "+123" would
// pass a check Rust rejects. Empty input returns true, matching Rust's all() on
// an empty iterator — the empty case is caught earlier by RequireNonemptySbd.
func allASCIIDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// ParseScores extracts subject scores from a DIEM_THI cell.
//
// Every one of the 16 patterns runs against every dataset; a subject absent from
// a given exam year never matches and stays NULL. Matching is unanchored
// first-match, like Rust's Regex::captures.
func ParseScores(diemThi string) map[string]float64 {
	out := make(map[string]float64)
	if diemThi == "" {
		return out
	}
	for field, re := range schema.ScorePatterns {
		m := re.FindStringSubmatch(diemThi)
		if m == nil {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		// Unreachable given the pattern shape, but kept for parity with
		// transform.rs:136 (is_finite).
		if math.IsInf(v, 0) || math.IsNaN(v) {
			continue
		}
		out[field] = v
	}
	return out
}

// ErrNoColumns is returned when the fixed-column path is used on a config that
// has no columns: mapping. Rust panics here via .expect() (transform.rs:162);
// returning an error is the Go-idiomatic equivalent and is unreachable in
// practice, since only the non-2016 path calls this.
var ErrNoColumns = errors.New("transform: config has no columns mapping")

// TransformRow extracts one row into a ParsedRow using fixed column indices,
// the 2017-family path. 2016 uses runtime format detection instead.
func TransformRow(raw []reader.Cell, cfg *config.DatasetConfig) (*ParsedRow, error) {
	cols := cfg.Columns
	if cols == nil {
		return nil, ErrNoColumns
	}

	// Trimmed accessor, mirroring the closure at transform.rs:163-167. Out-of-range
	// indices yield "" rather than an error, matching unwrap_or_default().
	get := func(idx int) string {
		if idx < 0 || idx >= len(raw) {
			return ""
		}
		return strings.TrimSpace(raw[idx].Str)
	}

	hoTen := get(cols.HoTen)
	ngaySinh := get(cols.NgaySinh)
	soBaoDanh := get(cols.SoBaoDanh)

	// diem_thi is read WITHOUT trimming (transform.rs:172-175), unlike the three
	// fields above. Harmless because the score patterns are unanchored, but it is
	// the shipped behaviour — do not "tidy" it.
	diemThi := ""
	if cols.DiemThi >= 0 && cols.DiemThi < len(raw) {
		diemThi = raw[cols.DiemThi].Str
	}

	var ngaySinhOpt *string
	if ngaySinh != "" {
		ngaySinhOpt = &ngaySinh
	}

	return &ParsedRow{
		SoBaoDanh:  soBaoDanh,
		HoTen:      hoTen,
		HoTenAscii: ToAscii(hoTen),
		NgaySinh:   ngaySinhOpt,
		TenCumThi:  nil,
		GioiTinh:   nil,
		Scores:     ParseScores(diemThi),
	}, nil
}
