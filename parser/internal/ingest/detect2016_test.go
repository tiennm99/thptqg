package ingest

import (
	"testing"

	"github.com/tiennm99/thptqg/parser/internal/reader"
)

// Ports the 11 tests in parser/src/format_detect_2016.rs, plus a guard per quirk.
//
// Phase 1 settled the Data -> string translation these fixtures need, so no
// guessing: Data::String(s) is s verbatim, Data::Float(8.0) renders "8" (Rust's
// f64 Display drops the .0, matching Go's FormatFloat(v,'f',-1,64)),
// Data::Float(8.5) renders "8.5", and Data::Empty is "" with IsEmpty true.

// --- header detection ---

func TestIsHeaderRow2016(t *testing.T) {
	if !IsHeaderRow2016(cells("SBD", "HOTEN", "TOAN")) {
		t.Error("SBD header not detected")
	}
	if !IsHeaderRow2016(cells("sobaodanh", "x")) {
		t.Error("lowercase SOBAODANH not detected")
	}
	if IsHeaderRow2016(cells("Nguyen Van A", "01/01/2000")) {
		t.Error("data row wrongly detected as header")
	}
	// The 2016 guard is len < 2, unlike the 2017 check's len < 3.
	if IsHeaderRow2016(cells("SBD")) {
		t.Error("single-cell row must not be a header")
	}
	if !IsHeaderRow2016(cells("SBD", "x")) {
		t.Error("two-cell row with a header token must be a header")
	}
}

// TestKnownHeadersHasTrailingSpaceToken pins the "SINH " literal. Trimming it
// would silently change which rows count as headers.
func TestKnownHeadersHasTrailingSpaceToken(t *testing.T) {
	var found bool
	for _, h := range KnownHeaders {
		if h == "SINH " {
			found = true
		}
	}
	if !found {
		t.Error(`KnownHeaders must contain "SINH " WITH its trailing space`)
	}
	if len(KnownHeaders) != 17 {
		t.Errorf("KnownHeaders has %d tokens, want 17", len(KnownHeaders))
	}
}

// --- format detection ---

func TestDetectFormatSeparateScores(t *testing.T) {
	f := DetectFormat(cells("SBD", "HOTEN", "TOAN", "VAN"))
	if f.Kind != FormatSeparateScores {
		t.Errorf("Kind = %v, want FormatSeparateScores", f.Kind)
	}
}

func TestDetectFormatMapped(t *testing.T) {
	f := DetectFormat(cells("STT", "SOBAODANH", "HO_TEN", "NGAY_SINH", "TEN_CUMTHI", "GIOI_TINH", "DIEM_THI"))
	if f.Kind != FormatMapped {
		t.Fatalf("Kind = %v, want FormatMapped", f.Kind)
	}
	if f.Sbd != 1 || f.HoTen != 2 || f.DiemThi != 6 {
		t.Errorf("indices sbd=%d ho_ten=%d diem_thi=%d", f.Sbd, f.HoTen, f.DiemThi)
	}
	if f.NgaySinh == nil || *f.NgaySinh != 3 || f.TenCumThi == nil || *f.TenCumThi != 4 || f.GioiTinh == nil || *f.GioiTinh != 5 {
		t.Error("optional indices not resolved")
	}
}

// TestDetectFormatMappedIsOrderIndependent: indices are resolved by name.
func TestDetectFormatMappedIsOrderIndependent(t *testing.T) {
	f := DetectFormat(cells("DIEM_THI", "HO_TEN", "SBD"))
	if f.Kind != FormatMapped || f.Sbd != 2 || f.DiemThi != 0 || f.HoTen != 1 {
		t.Errorf("got %+v", f)
	}
}

// TestDetectFormatMappedHoTenFallback ports the col-1 fallback
// (format_detect_2016.rs:132).
func TestDetectFormatMappedHoTenFallback(t *testing.T) {
	f := DetectFormat(cells("SBD", "SOMETHING", "DIEM_THI"))
	if f.Kind != FormatMapped {
		t.Fatalf("Kind = %v, want FormatMapped", f.Kind)
	}
	if f.HoTen != 1 {
		t.Errorf("ho_ten = %d, want fallback 1", f.HoTen)
	}
}

// TestDetectFormatDefault: a header lacking SBD or DIEM_THI falls back to the
// positional layout.
func TestDetectFormatDefault(t *testing.T) {
	f := DetectFormat(cells("A", "B", "C"))
	if f.Kind != FormatDefault {
		t.Errorf("Kind = %v, want FormatDefault", f.Kind)
	}
	if f.Sbd != 0 || f.HoTen != 1 || f.DiemThi != 5 {
		t.Errorf("default indices wrong: %+v", f)
	}
}

// --- row processing ---

func TestProcessSeparateScoresRow(t *testing.T) {
	// 0=SBD 1=HOTEN 2=TOAN 3=VAN 4=LY 5=HOA 6=SINH 7=SU 8=DIA 9,10=NN 11=NN total
	row := cells("1000", "Nguyễn Văn Đức", "8", "7.5", "", "6", "", "5", "", "", "", "9.25")
	got := ProcessRow2016(row, Format{Kind: FormatSeparateScores})
	if got == nil {
		t.Fatal("row rejected")
	}
	if got.SoBaoDanh != "1000" || got.HoTenAscii != "nguyen van duc" {
		t.Errorf("sbd=%q ascii=%q", got.SoBaoDanh, got.HoTenAscii)
	}
	want := map[string]float64{"toan": 8, "ngu_van": 7.5, "hoa_hoc": 6, "lich_su": 5, "tieng_anh": 9.25}
	if len(got.Scores) != len(want) {
		t.Errorf("scores = %v, want %v", got.Scores, want)
	}
	for k, v := range want {
		if got.Scores[k] != v {
			t.Errorf("%s = %v, want %v", k, got.Scores[k], v)
		}
	}
	// These columns are structurally unreachable in this layout.
	for _, absent := range []string{"tieng_phap", "tieng_duc", "tieng_nhat", "tieng_trung", "khtn", "khxh", "gdcd"} {
		if _, ok := got.Scores[absent]; ok {
			t.Errorf("%s must be unreachable in separate-scores", absent)
		}
	}
	if got.NgaySinh != nil || got.TenCumThi != nil || got.GioiTinh != nil {
		t.Error("ngay_sinh/ten_cum_thi/gioi_tinh are always nil in separate-scores")
	}
}

// TestZeroScoreBecomesNull pins the JS falsy quirk: parseFloat(x) || null means
// a literal 0 is indistinguishable from "no score" (format_detect_2016.rs:165).
func TestZeroScoreBecomesNull(t *testing.T) {
	row := cells("1000", "A", "0", "0.0", "1", "", "", "", "", "", "", "")
	got := ProcessRow2016(row, Format{Kind: FormatSeparateScores})
	if got == nil {
		t.Fatal("row rejected")
	}
	if _, ok := got.Scores["toan"]; ok {
		t.Error(`a "0" score must become NULL, not 0`)
	}
	if _, ok := got.Scores["ngu_van"]; ok {
		t.Error(`a "0.0" score must become NULL, not 0`)
	}
	if got.Scores["vat_ly"] != 1 {
		t.Error("a non-zero score must survive")
	}
}

// TestGenderAllowlist ports format_detect_2016.rs:263-271 — exactly two values.
func TestGenderAllowlist(t *testing.T) {
	f := defaultFormat()
	for in, want := range map[string]string{"Nam": "Nam", "Nữ": "Nữ"} {
		row := cells("1", "A", "", "", in, "")
		got := ProcessRow2016(row, f)
		if got == nil || got.GioiTinh == nil || *got.GioiTinh != want {
			t.Errorf("gioi_tinh for %q not preserved", in)
		}
	}
	for _, in := range []string{"nam", "NAM", "Unknown", "M", "F", "nữ", ""} {
		row := cells("1", "A", "", "", in, "")
		got := ProcessRow2016(row, f)
		if got == nil {
			t.Fatalf("row rejected for %q", in)
		}
		if got.GioiTinh != nil {
			t.Errorf("gioi_tinh for %q = %q, want nil", in, *got.GioiTinh)
		}
	}
}

// TestLeakedHeaderRowSkipped ports format_detect_2016.rs:244-250.
func TestLeakedHeaderRowSkipped(t *testing.T) {
	f := defaultFormat()
	if ProcessRow2016(cells("SBD", "HO_TEN", "", "", "", ""), f) != nil {
		t.Error("a repeated header row must be skipped")
	}
	if ProcessRow2016(cells("123", "SOBAODANH", "", "", "", ""), f) != nil {
		t.Error("a row whose name cell is a header token must be skipped")
	}
	if ProcessRow2016(cells("123", "Nguyen Van A", "", "", "", ""), f) == nil {
		t.Error("a genuine data row must not be skipped")
	}
}

// TestMappedRowEmptyFieldsRejected: either identity field empty rejects the row.
func TestMappedRowEmptyFieldsRejected(t *testing.T) {
	f := defaultFormat()
	if ProcessRow2016(cells("", "A", "", "", "", ""), f) != nil {
		t.Error("empty sbd must reject")
	}
	if ProcessRow2016(cells("1", "", "", "", "", ""), f) != nil {
		t.Error("empty ho_ten must reject")
	}
}

// TestMappedRowPopulates2016OnlyColumns: this is the only dataset that fills
// ten_cum_thi and gioi_tinh.
func TestMappedRowPopulates2016OnlyColumns(t *testing.T) {
	row := cells("123", "Lê Văn Long", "01/01/1998", "Cụm thi số 1", "Nam", "Toán: 7.5  Tiếng Đức: 6")
	got := ProcessRow2016(row, defaultFormat())
	if got == nil {
		t.Fatal("row rejected")
	}
	if got.NgaySinh == nil || *got.NgaySinh != "01/01/1998" {
		t.Errorf("ngay_sinh = %v", got.NgaySinh)
	}
	if got.TenCumThi == nil || *got.TenCumThi != "Cụm thi số 1" {
		t.Errorf("ten_cum_thi = %v", got.TenCumThi)
	}
	if got.Scores["toan"] != 7.5 || got.Scores["tieng_duc"] != 6 {
		t.Errorf("scores = %v", got.Scores)
	}
}

// TestDefaultIsMappedWithFixedIndices: FormatDefault must not be a separate code
// path (format_detect_2016.rs:295-306).
func TestDefaultIsMappedWithFixedIndices(t *testing.T) {
	row := cells("123", "A", "01/01/2000", "Cluster", "Nữ", "Toán: 5")
	viaDefault := ProcessRow2016(row, defaultFormat())
	two, three, four := 2, 3, 4
	viaMapped := ProcessRow2016(row, Format{
		Kind: FormatMapped, Sbd: 0, HoTen: 1,
		NgaySinh: &two, TenCumThi: &three, GioiTinh: &four, DiemThi: 5,
	})
	if viaDefault == nil || viaMapped == nil {
		t.Fatal("row rejected")
	}
	if viaDefault.SoBaoDanh != viaMapped.SoBaoDanh ||
		*viaDefault.TenCumThi != *viaMapped.TenCumThi ||
		*viaDefault.GioiTinh != *viaMapped.GioiTinh ||
		viaDefault.Scores["toan"] != viaMapped.Scores["toan"] {
		t.Error("FormatDefault must behave identically to the equivalent FormatMapped")
	}
}

// TestShortRowGuardIsSeparateFromValidation documents that rows under 2 cells
// are dropped by the loop before the counter (main.rs:351-353), not here.
func TestShortRowGuardIsSeparateFromValidation(t *testing.T) {
	if got := ProcessRow2016([]reader.Cell{{Str: "123"}}, defaultFormat()); got != nil {
		t.Error("a 1-cell row has no name and must be rejected")
	}
}
