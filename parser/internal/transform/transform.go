// Package transform performs row transformation: ASCII normalisation, score
// regex parsing, and validation.
package transform

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/tiennm99/thptqg/parser/internal/config"
	"github.com/tiennm99/thptqg/parser/internal/reader"
	"github.com/tiennm99/thptqg/parser/internal/schema"
)

// ToAscii normalises a Vietnamese name to an ASCII slug.
//
//  1. NFD decompose (splits base + combining diacritics)
//  2. Drop combining marks in U+0300..U+036F
//  3. Replace đ/Đ with d (NFD does not decompose them)
//  4. Lowercase
//
// Step 2 filters a LITERAL CODEPOINT RANGE, not a Unicode category.
// unicode.Is(unicode.Mn, r) is strictly broader and would strip marks this keeps,
// silently changing ho_ten_ascii — the column the site's accent-insensitive
// search runs on. That search normalises the query with the same four steps in
// web/src/App.jsx toAscii; the two must produce identical output or lookups miss.
func ToAscii(s string) string {
	if s == "" {
		return ""
	}
	decomposed := norm.NFD.String(s)

	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if r >= 0x0300 && r <= 0x036F {
			continue // combining mark inside the dropped range
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
// The distinction is load-bearing for the printed counters: the two non-blank
// reasons count as source rows while SkipBlankRow does not. That split lives in
// the CALLER — the build loop drops blank rows before the source-row counter.
type SkipReason int

const (
	SkipNone SkipReason = iota
	// SkipBlankRow: row is fully blank, and strip_blank_rows is on. Checked
	// before the source-row counter.
	SkipBlankRow
	// SkipEmptyField: so_bao_danh or ho_ten empty/missing.
	SkipEmptyField
	// SkipNonNumericSbd: so_bao_danh contains non-digit characters, and
	// require_numeric_sbd is on.
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
// stripBlankRows and allBlank are passed explicitly because a shorter signature
// could not express both blank-row paths.
func ValidateRow(hoTen, soBaoDanh string, cfg *config.ValidationCfg, stripBlankRows, allBlank bool) SkipReason {
	// Skip fully blank rows BEFORE counting source rows.
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

// allASCIIDigits reports whether every byte of s is an ASCII digit.
//
// Deliberately not strconv.Atoi: Atoi accepts a leading sign, so "+123" would
// pass a check that must reject it. Empty input returns true; the empty case is
// caught earlier by RequireNonemptySbd.
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
// first-match.
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
		// Unreachable given the pattern shape; kept so a widened pattern can
		// never write Inf or NaN into a score column.
		if math.IsInf(v, 0) || math.IsNaN(v) {
			continue
		}
		out[field] = v
	}
	return out
}

// ErrNoColumns is returned when the fixed-column path is used on a config that
// has no columns: mapping. Unreachable in practice, since only the non-2016 path
// calls TransformRow.
var ErrNoColumns = errors.New("transform: config has no columns mapping")

// TransformRow extracts one row into a ParsedRow using fixed column indices,
// the 2017 path. 2016 uses runtime format detection instead.
func TransformRow(raw []reader.Cell, cfg *config.DatasetConfig) (*ParsedRow, error) {
	cols := cfg.Columns
	if cols == nil {
		return nil, ErrNoColumns
	}

	// Trimmed accessor. Out-of-range indices yield "" rather than an error, so a
	// short row produces empty fields instead of failing.
	get := func(idx int) string {
		if idx < 0 || idx >= len(raw) {
			return ""
		}
		return strings.TrimSpace(raw[idx].Str)
	}

	hoTen := get(cols.HoTen)
	ngaySinh := get(cols.NgaySinh)
	soBaoDanh := get(cols.SoBaoDanh)
	diemThi := get(cols.DiemThi)

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
