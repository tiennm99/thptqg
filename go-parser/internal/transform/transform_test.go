package transform

import (
	"testing"

	"github.com/tiennm99/thptqg/go-parser/internal/config"
	"github.com/tiennm99/thptqg/go-parser/internal/reader"
)

// Ports every test in parser/src/transform.rs's test module (:201-409) — 29 in
// total, not the 20 in the :213-315 range, which covers only ToAscii. The nine
// outside that range are the ParseScores and ValidateRow cases, i.e. exactly the
// behaviours this package's traps concern.

// --- ToAscii: the 20 cases at transform.rs:213-315 ---

func TestToAscii(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain_latin", "Nguyen Van A", "nguyen van a"},
		{"nguyen_thi_hoa", "Nguyễn Thị Hoa", "nguyen thi hoa"},
		{"tran_van_duc", "Trần Văn Đức", "tran van duc"},
		{"le_thi_my_duyen", "Lê Thị Mỹ Duyên", "le thi my duyen"},
		{"pham_thi_lan", "Phạm Thị Lan", "pham thi lan"},
		{"bui_thi_thu", "Bùi Thị Thu", "bui thi thu"},
		{"hoang_van_truong", "Hoàng Văn Trường", "hoang van truong"},
		{"do_thi_ngan", "Đỗ Thị Ngân", "do thi ngan"},
		{"nguyen_van_khanh", "Nguyễn Văn Khánh", "nguyen van khanh"},
		{"trinh_thi_bich_ngoc", "Trịnh Thị Bích Ngọc", "trinh thi bich ngoc"},
		{"vu_thi_dieu", "Vũ Thị Diệu", "vu thi dieu"},
		{"nguyen_thi_tuong_vi", "Nguyễn Thị Tường Vi", "nguyen thi tuong vi"},
		{"lowercase_d_stroke", "đặng thị hằng", "dang thi hang"},
		{"uppercase_d_stroke", "ĐẶNG THỊ HẰNG", "dang thi hang"},
		{"mixed_case", "NGUYỄN VĂN AN", "nguyen van an"},
		{"tran_thi_kim_anh", "Trần Thị Kim Anh", "tran thi kim anh"},
		{"nguyen_thi_phuong_thao", "Nguyễn Thị Phương Thảo", "nguyen thi phuong thao"},
		{"le_van_long", "Lê Văn Long", "le van long"},
		{"vo_thi_xuan_mai", "Võ Thị Xuân Mai", "vo thi xuan mai"},
		{"empty_string", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ToAscii(c.in); got != c.want {
				t.Errorf("ToAscii(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestToAsciiUsesLiteralRangeNotUnicodeMn guards the highest-value trap in this
// package. transform.rs:56 filters the literal range U+0300..U+036F; the inline
// comment at :53 calls it "Unicode category M", but the code is the spec.
// unicode.Mn is strictly broader, so using it would strip marks Rust keeps.
// U+0654 (ARABIC HAMZA ABOVE) is in Mn but outside the range: Rust keeps it.
func TestToAsciiUsesLiteralRangeNotUnicodeMn(t *testing.T) {
	const in = "aٔb"
	if got := ToAscii(in); got != in {
		t.Errorf("ToAscii(%q) = %q — a combining mark outside U+0300..U+036F must survive; "+
			"stripping it means unicode.Mn was used instead of the literal range", in, got)
	}
	// And a mark inside the range must be stripped.
	if got := ToAscii("áb"); got != "ab" {
		t.Errorf("ToAscii(\"a\\u0301b\") = %q, want \"ab\"", got)
	}
}

// TestToAsciiDStrokeIndependentOfNFD proves the đ/Đ replacement is a separate
// step: NFD does not decompose them, so relying on the mark filter alone loses
// the letter entirely.
func TestToAsciiDStrokeIndependentOfNFD(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"đ", "d"}, {"Đ", "d"}, {"đĐ", "dd"},
	} {
		if got := ToAscii(c.in); got != c.want {
			t.Errorf("ToAscii(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- ParseScores: transform.rs:323, :332, :342 ---

func TestParseScoresSingle(t *testing.T) {
	s := ParseScores("Toán: 8.5")
	if v, ok := s["toan"]; !ok || v != 8.5 {
		t.Errorf("toan = %v (present=%v), want 8.5", v, ok)
	}
	if _, ok := s["ngu_van"]; ok {
		t.Error("ngu_van should be absent")
	}
}

func TestParseScoresMultiple(t *testing.T) {
	s := ParseScores("Toán: 7.25  Ngữ văn: 6.0  Vật lí: 9")
	for field, want := range map[string]float64{"toan": 7.25, "ngu_van": 6.0, "vat_ly": 9.0} {
		if v, ok := s[field]; !ok || v != want {
			t.Errorf("%s = %v (present=%v), want %v", field, v, ok, want)
		}
	}
}

func TestParseScoresEmptyCell(t *testing.T) {
	if s := ParseScores(""); len(s) != 0 {
		t.Errorf("ParseScores(\"\") = %v, want empty", s)
	}
}

// TestParseScoresRealCellShape uses the wide space runs seen in the corpus.
func TestParseScoresRealCellShape(t *testing.T) {
	const cell = "Toán:   4.60   Ngữ văn:   5.50   Lịch sử:   4.50   "
	s := ParseScores(cell)
	if len(s) != 3 {
		t.Fatalf("matched %d subjects, want 3: %v", len(s), s)
	}
	if s["toan"] != 4.60 || s["ngu_van"] != 5.50 || s["lich_su"] != 4.50 {
		t.Errorf("got %v", s)
	}
}

// --- ValidateRow: transform.rs:359, :365, :374, :383, :393, :400 ---

func defaultValidation() *config.ValidationCfg {
	return &config.ValidationCfg{
		RequireNumericSbd:   false,
		RequireNonemptyName: true,
		RequireNonemptySbd:  true,
	}
}

func TestValidateOK(t *testing.T) {
	if r := ValidateRow("Nguyen Van A", "12345678", defaultValidation(), false, false); r != SkipNone {
		t.Errorf("got %v, want SkipNone", r)
	}
}

func TestValidateEmptySbd(t *testing.T) {
	if r := ValidateRow("Nguyen Van A", "", defaultValidation(), false, false); r != SkipEmptyField {
		t.Errorf("got %v, want SkipEmptyField", r)
	}
}

func TestValidateEmptyName(t *testing.T) {
	if r := ValidateRow("", "12345678", defaultValidation(), false, false); r != SkipEmptyField {
		t.Errorf("got %v, want SkipEmptyField", r)
	}
}

func TestValidateNonNumericSbdRejected(t *testing.T) {
	v := defaultValidation()
	v.RequireNumericSbd = true
	if r := ValidateRow("Nguyen Van A", "12AB5678", v, false, false); r != SkipNonNumericSbd {
		t.Errorf("got %v, want SkipNonNumericSbd", r)
	}
}

func TestValidateNumericSbdAccepted(t *testing.T) {
	v := defaultValidation()
	v.RequireNumericSbd = true
	if r := ValidateRow("Nguyen Van A", "12345678", v, false, false); r != SkipNone {
		t.Errorf("got %v, want SkipNone", r)
	}
}

func TestValidateBlankRowSkipped(t *testing.T) {
	if r := ValidateRow("", "", defaultValidation(), true, true); r != SkipBlankRow {
		t.Errorf("got %v, want SkipBlankRow", r)
	}
}

// TestValidateNumericSbdIsDigitScanNotAtoi: Rust uses chars().all(is_ascii_digit),
// which strconv.Atoi does not reproduce — Atoi accepts a leading sign, and would
// wrongly admit "+123".
func TestValidateNumericSbdIsDigitScanNotAtoi(t *testing.T) {
	v := defaultValidation()
	v.RequireNumericSbd = true
	for _, sbd := range []string{"+123", "-123", "12 3", "1.0", "ABC123", "１２３"} {
		if r := ValidateRow("Nguyen Van A", sbd, v, false, false); r != SkipNonNumericSbd {
			t.Errorf("ValidateRow(sbd=%q) = %v, want SkipNonNumericSbd", sbd, r)
		}
	}
}

// TestValidateBlankRowOnlyWhenStripEnabled: with strip_blank_rows false, an
// all-blank row falls through to the empty-field checks instead (transform.rs:109).
func TestValidateBlankRowOnlyWhenStripEnabled(t *testing.T) {
	if r := ValidateRow("", "", defaultValidation(), false, true); r != SkipEmptyField {
		t.Errorf("got %v, want SkipEmptyField when strip_blank_rows is off", r)
	}
}

// --- TransformRow ---

func fixedColumnCfg() *config.DatasetConfig {
	return &config.DatasetConfig{
		Columns:    &config.ColumnMap{HoTen: 0, NgaySinh: 1, SoBaoDanh: 2, DiemThi: 3},
		Validation: *defaultValidation(),
	}
}

func cells(vals ...string) []reader.Cell {
	out := make([]reader.Cell, len(vals))
	for i, v := range vals {
		out[i] = reader.Cell{Str: v, IsEmpty: v == ""}
	}
	return out
}

func TestTransformRow(t *testing.T) {
	row := cells("Nguyễn Văn Đức", "04/04/1999", "51002167", "Toán: 8.5  Ngữ văn: 7")
	got, err := TransformRow(row, fixedColumnCfg())
	if err != nil {
		t.Fatalf("TransformRow: %v", err)
	}
	if got.HoTen != "Nguyễn Văn Đức" || got.HoTenAscii != "nguyen van duc" {
		t.Errorf("ho_ten=%q ascii=%q", got.HoTen, got.HoTenAscii)
	}
	if got.SoBaoDanh != "51002167" {
		t.Errorf("so_bao_danh = %q", got.SoBaoDanh)
	}
	if got.NgaySinh == nil || *got.NgaySinh != "04/04/1999" {
		t.Errorf("ngay_sinh = %v", got.NgaySinh)
	}
	// 2016-only columns are never populated on the fixed-column path.
	if got.TenCumThi != nil || got.GioiTinh != nil {
		t.Error("ten_cum_thi and gioi_tinh must stay nil on the 2017 path")
	}
	if got.Scores["toan"] != 8.5 || got.Scores["ngu_van"] != 7 {
		t.Errorf("scores = %v", got.Scores)
	}
}

// TestTransformRowEmptyNgaySinhBecomesNil ports transform.rs:179-183.
func TestTransformRowEmptyNgaySinhBecomesNil(t *testing.T) {
	got, err := TransformRow(cells("A", "", "1", ""), fixedColumnCfg())
	if err != nil {
		t.Fatalf("TransformRow: %v", err)
	}
	if got.NgaySinh != nil {
		t.Errorf("empty ngay_sinh should be nil, got %q", *got.NgaySinh)
	}
}

// TestTransformRowShortRowYieldsEmptyFields ports the unwrap_or_default()
// behaviour at transform.rs:163-176: a row shorter than the configured indices
// yields empty strings rather than an error.
func TestTransformRowShortRowYieldsEmptyFields(t *testing.T) {
	got, err := TransformRow(cells("OnlyName"), fixedColumnCfg())
	if err != nil {
		t.Fatalf("TransformRow: %v", err)
	}
	if got.HoTen != "OnlyName" || got.SoBaoDanh != "" || got.NgaySinh != nil {
		t.Errorf("got ho_ten=%q sbd=%q ngay_sinh=%v", got.HoTen, got.SoBaoDanh, got.NgaySinh)
	}
}

// TestTransformRowDiemThiIsNotTrimmed pins an asymmetry that is easy to
// "tidy away": ho_ten, ngay_sinh and so_bao_danh are trimmed through the closure
// at transform.rs:164-168, but diem_thi is read raw at :172-175.
func TestTransformRowDiemThiIsNotTrimmed(t *testing.T) {
	row := cells("  A  ", "  01/01/2000  ", "  123  ", "   Toán: 5   ")
	got, err := TransformRow(row, fixedColumnCfg())
	if err != nil {
		t.Fatalf("TransformRow: %v", err)
	}
	if got.HoTen != "A" || got.SoBaoDanh != "123" {
		t.Errorf("trimmed fields wrong: ho_ten=%q sbd=%q", got.HoTen, got.SoBaoDanh)
	}
	if got.NgaySinh == nil || *got.NgaySinh != "01/01/2000" {
		t.Errorf("ngay_sinh = %v, want trimmed", got.NgaySinh)
	}
	// Untrimmed diem_thi still parses — the regexes are unanchored.
	if got.Scores["toan"] != 5 {
		t.Errorf("scores = %v", got.Scores)
	}
}

// TestTransformRowRequiresColumns: the fixed-column path is only reachable when
// the config has a columns: mapping. Rust panics via .expect() (transform.rs:162);
// Go returns an error instead.
func TestTransformRowRequiresColumns(t *testing.T) {
	cfg := &config.DatasetConfig{Validation: *defaultValidation()}
	if _, err := TransformRow(cells("A", "B", "C", "D"), cfg); err == nil {
		t.Fatal("TransformRow without a columns: mapping must return an error")
	}
}
