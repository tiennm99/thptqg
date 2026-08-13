// Vietnamese university admission-block ("khối thi") definitions.
// Each block is the sum of 3 subjects; we only compute a block when ALL three
// subject scores exist for the student.
//
// One list serves both exam years. computeBlocks() skips any block with a
// missing subject score, so blocks needing GDCD self-exclude on 2016 rows and
// blocks needing Tiếng Đức / Nhật self-exclude wherever those weren't sat —
// no per-year branching required.
//
// Based on Circular 03/2017/TT-BGDĐT.
export const ADMISSION_BLOCKS = [
  { code: "A00", subjects: ["toan", "vat_ly", "hoa_hoc"],     label: "Toán + Lý + Hóa" },
  { code: "A01", subjects: ["toan", "vat_ly", "tieng_anh"],   label: "Toán + Lý + Anh" },
  { code: "A02", subjects: ["toan", "vat_ly", "sinh_hoc"],    label: "Toán + Lý + Sinh" },
  { code: "A03", subjects: ["toan", "vat_ly", "lich_su"],     label: "Toán + Lý + Sử" },
  { code: "A04", subjects: ["toan", "vat_ly", "dia_ly"],      label: "Toán + Lý + Địa" },
  { code: "A05", subjects: ["toan", "hoa_hoc", "lich_su"],    label: "Toán + Hóa + Sử" },
  { code: "A06", subjects: ["toan", "hoa_hoc", "dia_ly"],     label: "Toán + Hóa + Địa" },
  { code: "A07", subjects: ["toan", "lich_su", "dia_ly"],     label: "Toán + Sử + Địa" },
  { code: "A08", subjects: ["toan", "lich_su", "gdcd"],       label: "Toán + Sử + GDCD" },
  { code: "A09", subjects: ["toan", "dia_ly", "gdcd"],        label: "Toán + Địa + GDCD" },
  { code: "A10", subjects: ["toan", "vat_ly", "gdcd"],        label: "Toán + Lý + GDCD" },
  { code: "A11", subjects: ["toan", "hoa_hoc", "gdcd"],       label: "Toán + Hóa + GDCD" },
  { code: "B00", subjects: ["toan", "hoa_hoc", "sinh_hoc"],   label: "Toán + Hóa + Sinh" },
  { code: "B01", subjects: ["toan", "sinh_hoc", "lich_su"],   label: "Toán + Sinh + Sử" },
  { code: "B02", subjects: ["toan", "sinh_hoc", "dia_ly"],    label: "Toán + Sinh + Địa" },
  { code: "B03", subjects: ["toan", "sinh_hoc", "ngu_van"],   label: "Toán + Sinh + Văn" },
  { code: "B04", subjects: ["toan", "sinh_hoc", "gdcd"],      label: "Toán + Sinh + GDCD" },
  { code: "B08", subjects: ["toan", "sinh_hoc", "tieng_anh"], label: "Toán + Sinh + Anh" },
  { code: "C00", subjects: ["ngu_van", "lich_su", "dia_ly"],  label: "Văn + Sử + Địa" },
  { code: "C01", subjects: ["ngu_van", "toan", "vat_ly"],     label: "Văn + Toán + Lý" },
  { code: "C02", subjects: ["ngu_van", "toan", "hoa_hoc"],    label: "Văn + Toán + Hóa" },
  { code: "C03", subjects: ["ngu_van", "toan", "lich_su"],    label: "Văn + Toán + Sử" },
  { code: "C04", subjects: ["ngu_van", "toan", "dia_ly"],     label: "Văn + Toán + Địa" },
  { code: "C05", subjects: ["ngu_van", "vat_ly", "hoa_hoc"],  label: "Văn + Lý + Hóa" },
  { code: "C06", subjects: ["ngu_van", "vat_ly", "sinh_hoc"], label: "Văn + Lý + Sinh" },
  { code: "C07", subjects: ["ngu_van", "vat_ly", "lich_su"],  label: "Văn + Lý + Sử" },
  { code: "C08", subjects: ["ngu_van", "hoa_hoc", "sinh_hoc"], label: "Văn + Hóa + Sinh" },
  { code: "C09", subjects: ["ngu_van", "vat_ly", "dia_ly"],   label: "Văn + Lý + Địa" },
  { code: "C10", subjects: ["ngu_van", "hoa_hoc", "lich_su"], label: "Văn + Hóa + Sử" },
  { code: "C12", subjects: ["ngu_van", "sinh_hoc", "lich_su"], label: "Văn + Sinh + Sử" },
  { code: "C13", subjects: ["ngu_van", "sinh_hoc", "dia_ly"], label: "Văn + Sinh + Địa" },
  { code: "C14", subjects: ["ngu_van", "toan", "gdcd"],       label: "Văn + Toán + GDCD" },
  { code: "C16", subjects: ["ngu_van", "vat_ly", "gdcd"],     label: "Văn + Lý + GDCD" },
  { code: "C17", subjects: ["ngu_van", "hoa_hoc", "gdcd"],    label: "Văn + Hóa + GDCD" },
  { code: "C19", subjects: ["ngu_van", "lich_su", "gdcd"],    label: "Văn + Sử + GDCD" },
  { code: "C20", subjects: ["ngu_van", "dia_ly", "gdcd"],     label: "Văn + Địa + GDCD" },
  { code: "D01", subjects: ["toan", "ngu_van", "tieng_anh"],  label: "Toán + Văn + Anh" },
  { code: "D02", subjects: ["toan", "ngu_van", "tieng_nga"],  label: "Toán + Văn + Nga" },
  { code: "D03", subjects: ["toan", "ngu_van", "tieng_phap"], label: "Toán + Văn + Pháp" },
  { code: "D04", subjects: ["toan", "ngu_van", "tieng_trung"], label: "Toán + Văn + Trung" },
  { code: "D07", subjects: ["toan", "hoa_hoc", "tieng_anh"],  label: "Toán + Hóa + Anh" },
  { code: "D08", subjects: ["toan", "sinh_hoc", "tieng_anh"], label: "Toán + Sinh + Anh" },
  { code: "D09", subjects: ["toan", "lich_su", "tieng_anh"],  label: "Toán + Sử + Anh" },
  { code: "D10", subjects: ["toan", "dia_ly", "tieng_anh"],   label: "Toán + Địa + Anh" },
  { code: "D11", subjects: ["ngu_van", "vat_ly", "tieng_anh"], label: "Văn + Lý + Anh" },
  { code: "D12", subjects: ["ngu_van", "hoa_hoc", "tieng_anh"], label: "Văn + Hóa + Anh" },
  { code: "D13", subjects: ["ngu_van", "sinh_hoc", "tieng_anh"], label: "Văn + Sinh + Anh" },
  { code: "D14", subjects: ["ngu_van", "lich_su", "tieng_anh"], label: "Văn + Sử + Anh" },
  { code: "D15", subjects: ["ngu_van", "dia_ly", "tieng_anh"], label: "Văn + Địa + Anh" },

  // German and Japanese blocks. Previously omitted because neither language
  // appeared in the 2017 database — but that was a parser gap, not reality:
  // unifying the subject patterns recovered German and Japanese scores in every
  // dataset. Students who sat neither are unaffected, since computeBlocks()
  // skips blocks with a missing subject.
  { code: "D05", subjects: ["toan", "ngu_van", "tieng_duc"],  label: "Toán + Văn + Đức" },
  { code: "D06", subjects: ["toan", "ngu_van", "tieng_nhat"], label: "Toán + Văn + Nhật" },
];

// Returns { code, label, total, parts:[{key,score}] } for every block where
// the student has scores for all three subjects, sorted by total desc.
export function computeBlocks(student) {
  const out = [];
  for (const b of ADMISSION_BLOCKS) {
    const parts = b.subjects.map((k) => ({ key: k, score: student[k] }));
    if (parts.some((p) => p.score === null || p.score === undefined)) continue;
    const total = parts.reduce((s, p) => s + p.score, 0);
    out.push({ code: b.code, label: b.label, total, parts });
  }
  return out.sort((a, b) => b.total - a.total);
}

// Score tier: TFT rarity-style ladder. 6 stops match the Vietnamese exam
// reality (≤1 is "điểm liệt" — automatic fail regardless of other scores).
// Color is paired with a unicode symbol so meaning is never color-only.
export function scoreTier(score) {
  if (score === null || score === undefined) return null;
  if (score <= 1) return { key: "common",      symbol: "·", label: "Điểm liệt" };
  if (score < 5)  return { key: "uncommon",    symbol: "○", label: "Chưa đạt" };
  if (score < 6.5) return { key: "rare",       symbol: "◆", label: "Trung bình" };
  if (score < 8)  return { key: "epic",        symbol: "★", label: "Khá" };
  if (score < 9)  return { key: "legendary",   symbol: "✦", label: "Giỏi" };
  return { key: "prismatic",                    symbol: "❖", label: "Xuất sắc" };
}
