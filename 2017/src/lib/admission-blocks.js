// Vietnamese university admission-block ("khối thi") definitions for 2017.
// Each block is the sum of 3 subjects; we only compute a block when ALL three
// subject scores exist for the student.

export const ADMISSION_BLOCKS = [
  { code: "A",  subjects: ["toan", "vat_ly", "hoa_hoc"],    label: "Toán + Lý + Hóa" },
  { code: "A1", subjects: ["toan", "vat_ly", "tieng_anh"],  label: "Toán + Lý + Anh" },
  { code: "B",  subjects: ["toan", "hoa_hoc", "sinh_hoc"],  label: "Toán + Hóa + Sinh" },
  { code: "C",  subjects: ["ngu_van", "lich_su", "dia_ly"], label: "Văn + Sử + Địa" },
  { code: "D1", subjects: ["toan", "ngu_van", "tieng_anh"], label: "Toán + Văn + Anh" },
  { code: "D2", subjects: ["toan", "ngu_van", "tieng_nga"], label: "Toán + Văn + Nga" },
  { code: "D3", subjects: ["toan", "ngu_van", "tieng_phap"], label: "Toán + Văn + Pháp" },
  { code: "D4", subjects: ["toan", "ngu_van", "tieng_trung"], label: "Toán + Văn + Trung" },
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

// Score tier: used for color + icon coding (meaning never conveyed by color alone)
export function scoreTier(score) {
  if (score === null || score === undefined) return null;
  if (score < 5) return { key: "poor",      symbol: "▽", label: "Chưa đạt" };
  if (score < 6.5) return { key: "weak",     symbol: "○", label: "Trung bình" };
  if (score < 8) return { key: "fair",       symbol: "◆", label: "Khá" };
  if (score < 9) return { key: "good",       symbol: "★", label: "Giỏi" };
  return { key: "excellent",                  symbol: "✦", label: "Xuất sắc" };
}
