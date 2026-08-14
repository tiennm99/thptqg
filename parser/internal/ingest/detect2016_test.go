package ingest

import (
	"strings"
	"testing"

	"github.com/tiennm99/thptqg/parser/internal/reader"
)

// Fixtures use the reader's rendering: a string cell is its text verbatim, a
// numeric cell is its shortest round-tripping form (8.0 is "8", 8.5 is "8.5"),
// and a cell with no value is "" with IsEmpty set.

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

// TestKnownHeadersAreTrimmed: the cell is trimmed before comparison, so a token
// carrying surrounding space could never match anything.
func TestKnownHeadersAreTrimmed(t *testing.T) {
	for _, h := range KnownHeaders {
		if h != strings.TrimSpace(h) {
			t.Errorf("KnownHeaders token %q has surrounding space and can never match", h)
		}
	}
}

// TestSheetFormatSkipsTitleBlock: one file opens with a ministry title block
// above its header row, and the rows above the header are not data.
func TestSheetFormatSkipsTitleBlock(t *testing.T) {
	rows := [][]reader.Cell{
		cells("BỘ GIÁO DỤC VÀ ĐÀO TẠO", "", ""),
		cells("ĐƠN VỊ:", "TRƯỜNG ĐẠI HỌC X", ""),
		cells("DANH SÁCH ĐIỂM THÍ SINH", "", ""),
		cells("STT", "SBD", "Họ tên", "Ngày sinh", "Giới tính", "CMND", "Tỉnh", "TO", "VA", "LI"),
		cells("1", "DCT000001", "Nguyễn Văn A", "01/01/1998", "Nam", "123", "46", "8", "7", "6"),
	}
	f, start := sheetFormat(rows)
	if start != 4 {
		t.Errorf("data starts at row %d, want 4", start)
	}
	if f.Kind != FormatSubjectColumns {
		t.Errorf("Kind = %v, want FormatSubjectColumns", f.Kind)
	}
}

// TestSheetFormatHeaderlessSheet: a sheet whose first row is already data keeps
// the positional layout and starts at row 0.
func TestSheetFormatHeaderlessSheet(t *testing.T) {
	rows := [][]reader.Cell{
		cells("YTB000001", "PHẠM THỊ HỒNG ÁI", "21/08/1998", "Y Dược Thái Bình", "Nữ", "Toán: 7.75"),
		cells("YTB000002", "TRẦN VĂN B", "22/08/1998", "Y Dược Thái Bình", "Nam", "Toán: 5"),
	}
	f, start := sheetFormat(rows)
	if start != 0 || f.Kind != FormatDefault {
		t.Errorf("got kind=%v start=%d, want FormatDefault at 0", f.Kind, start)
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

// TestDetectFormatMappedHoTenFallback covers the col-1 fallback for ho_ten.
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

// TestZeroScoreIsKept: a candidate who sat a paper and scored nothing has a
// score of 0, not a missing one.
func TestZeroScoreIsKept(t *testing.T) {
	row := cells("1000", "A", "0", "0.0", "1", "", "", "", "", "", "", "")
	got := ProcessRow2016(row, Format{Kind: FormatSeparateScores})
	if got == nil {
		t.Fatal("row rejected")
	}
	for field, want := range map[string]float64{"toan": 0, "ngu_van": 0, "vat_ly": 1} {
		v, ok := got.Scores[field]
		if !ok || v != want {
			t.Errorf("%s = %v (present=%v), want %v", field, v, ok, want)
		}
	}
	if _, ok := got.Scores["hoa_hoc"]; ok {
		t.Error("an empty cell must stay absent")
	}
}

// TestGenderNormalisation: "Nam"/"Nữ" verbatim, the Cần Thơ files' 1/0 encoding
// translated, everything else NULL.
func TestGenderNormalisation(t *testing.T) {
	f := defaultFormat()
	for in, want := range map[string]string{"Nam": "Nam", "Nữ": "Nữ", "1": "Nữ", "0": "Nam"} {
		row := cells("1", "A", "", "", in, "")
		got := ProcessRow2016(row, f)
		if got == nil || got.GioiTinh == nil || *got.GioiTinh != want {
			t.Errorf("gioi_tinh for %q = %v, want %q", in, got.GioiTinh, want)
		}
	}
	for _, in := range []string{"nam", "NAM", "Unknown", "M", "F", "nữ", "", "2"} {
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

// TestBirthDateExpandsCompactForm: ddmmyy becomes dd/mm/19yy so the column
// holds one format; anything else is passed through.
func TestBirthDateExpandsCompactForm(t *testing.T) {
	f := defaultFormat()
	for in, want := range map[string]string{
		"140798":     "14/07/1998",
		"01/01/1998": "01/01/1998",
		"1998":       "1998",
	} {
		got := ProcessRow2016(cells("1", "A", in, "", "", ""), f)
		if got == nil || got.NgaySinh == nil || *got.NgaySinh != want {
			t.Errorf("ngay_sinh for %q = %v, want %q", in, got.NgaySinh, want)
		}
	}
}

// --- per-subject score columns ---

// TestDetectFormatSubjectColumnsTwoLetter covers the two-letter subject columns,
// whose foreign-language score is filed under the subject its code names.
func TestDetectFormatSubjectColumnsTwoLetter(t *testing.T) {
	header := cells("STT", "SBD", "Họ tên", "Ngày sinh", "Giới tính", "CMND", "Tỉnh",
		"TO", "VA", "LI", "HO", "SI", "SU", "DI", "NN", "Môn NN")
	f := DetectFormat(header)
	if f.Kind != FormatSubjectColumns {
		t.Fatalf("Kind = %v, want FormatSubjectColumns", f.Kind)
	}
	if f.Sbd != 1 || f.HoTen != 2 {
		t.Errorf("identity: sbd=%d ho_ten=%d", f.Sbd, f.HoTen)
	}
	if f.NgaySinh == nil || *f.NgaySinh != 3 || f.GioiTinh == nil || *f.GioiTinh != 4 {
		t.Error("accented identity headers not resolved")
	}
	if f.Subjects["toan"] != 7 || f.Subjects["dia_ly"] != 13 {
		t.Errorf("subject columns = %v", f.Subjects)
	}

	row := cells("1", "DCT000073", "Trần Thị Phước An", "23/11/1998", "Nữ", "291183999", "46",
		"6.25", "4.5", "5.8", "", "", "", "", "5.38", "N1")
	got := ProcessRow2016(row, f)
	if got == nil {
		t.Fatal("row rejected")
	}
	if got.SoBaoDanh != "DCT000073" || got.HoTen != "Trần Thị Phước An" {
		t.Errorf("identity = %q / %q", got.SoBaoDanh, got.HoTen)
	}
	want := map[string]float64{"toan": 6.25, "ngu_van": 4.5, "vat_ly": 5.8, "tieng_anh": 5.38}
	if len(got.Scores) != len(want) {
		t.Errorf("scores = %v, want %v", got.Scores, want)
	}
	for k, v := range want {
		if got.Scores[k] != v {
			t.Errorf("%s = %v, want %v", k, got.Scores[k], v)
		}
	}
}

// TestDetectFormatSubjectColumnsCanTho covers the CDIEM<n> columns, which are
// numbered in exam-timetable order rather than named.
func TestDetectFormatSubjectColumnsCanTho(t *testing.T) {
	header := cells("sbd", "hoten", "ho", "ten", "phai", "ngaysinh", "socmnd",
		"cdiem1", "cdiem2", "cdiem3", "cdiem4", "cdiem5", "cdiem6", "cdiem7", "cdiem8", "ngoaingu")
	f := DetectFormat(header)
	if f.Kind != FormatSubjectColumns {
		t.Fatalf("Kind = %v, want FormatSubjectColumns", f.Kind)
	}
	// "ho" is the surname column here, and must not be read as hoa_hoc.
	if got, ok := f.Subjects["hoa_hoc"]; ok && got == 2 {
		t.Error("surname column resolved as a score column")
	}
	if f.Subjects["toan"] != 7 || f.Subjects["ngu_van"] != 9 || f.Subjects["sinh_hoc"] != 14 {
		t.Errorf("subject columns = %v", f.Subjects)
	}
	if f.LangScore == nil || *f.LangScore != 8 || f.LangCode == nil || *f.LangCode != 15 {
		t.Error("language score/code columns not resolved")
	}

	row := cells("TCT000001", "Dương Diễm ái", "Dương Diễm", "ái", "1", "140798", "362539228",
		"07.50", "05.60", "06.50", "", "", "08.20", "", "07.60", "N3")
	got := ProcessRow2016(row, f)
	if got == nil {
		t.Fatal("row rejected")
	}
	if got.GioiTinh == nil || *got.GioiTinh != "Nữ" {
		t.Errorf("gioi_tinh = %v, want Nữ", got.GioiTinh)
	}
	if got.NgaySinh == nil || *got.NgaySinh != "14/07/1998" {
		t.Errorf("ngay_sinh = %v", got.NgaySinh)
	}
	want := map[string]float64{"toan": 7.5, "tieng_phap": 5.6, "ngu_van": 6.5, "hoa_hoc": 8.2, "sinh_hoc": 7.6}
	if len(got.Scores) != len(want) {
		t.Errorf("scores = %v, want %v", got.Scores, want)
	}
	for k, v := range want {
		if got.Scores[k] != v {
			t.Errorf("%s = %v, want %v", k, got.Scores[k], v)
		}
	}
}

// TestSubjectColumnsUnknownLanguageCode: an unrecognised code drops the score
// rather than filing it under a guess.
func TestSubjectColumnsUnknownLanguageCode(t *testing.T) {
	header := cells("sbd", "hoten", "ho", "ten", "phai", "ngaysinh", "socmnd",
		"cdiem1", "cdiem2", "cdiem3", "cdiem4", "cdiem5", "cdiem6", "cdiem7", "cdiem8", "ngoaingu")
	f := DetectFormat(header)
	row := cells("TCT000002", "B", "", "", "0", "", "", "5", "6", "", "", "", "", "", "", "N9")
	got := ProcessRow2016(row, f)
	if got == nil {
		t.Fatal("row rejected")
	}
	if len(got.Scores) != 1 || got.Scores["toan"] != 5 {
		t.Errorf("scores = %v, want only toan", got.Scores)
	}
}

// TestSubjectColumnsNeedsThreeSubjects: a stray two-letter header elsewhere must
// not turn an unrelated file into this layout.
func TestSubjectColumnsNeedsThreeSubjects(t *testing.T) {
	f := DetectFormat(cells("SBD", "HOTEN", "DI", "SU"))
	if f.Kind == FormatSubjectColumns {
		t.Error("two subject columns must not be enough to select the layout")
	}
}

// TestLeakedHeaderRowSkipped: a header repeated inside the data must not become
// a student.
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

// TestDefaultIsMappedWithFixedIndices: FormatDefault must not become a separate
// code path.
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
// are dropped by the ingest loop before the counter, not by row processing.
func TestShortRowGuardIsSeparateFromValidation(t *testing.T) {
	if got := ProcessRow2016([]reader.Cell{{Str: "123"}}, defaultFormat()); got != nil {
		t.Error("a 1-cell row has no name and must be rejected")
	}
}
