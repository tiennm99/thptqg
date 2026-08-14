/**
 * The 16 subject columns of the canonical schema, in display order. Mirrors
 * `ScoreFields` in parser/internal/schema/schema.go, and is the single list both
 * score-table.jsx and student-detail.jsx render from.
 *
 * Nothing here is dataset-specific. Columns an exam year has no data for are
 * NULL on every row, and the UI hides all-NULL columns, so one list serves both
 * years without branching.
 */

export const SUBJECTS = [
  { key: "toan", label: "Toán" },
  { key: "ngu_van", label: "Ngữ văn" },
  { key: "vat_ly", label: "Vật lí" },
  { key: "hoa_hoc", label: "Hóa học" },
  { key: "sinh_hoc", label: "Sinh học" },
  { key: "khtn", label: "KHTN" },
  { key: "lich_su", label: "Lịch sử" },
  { key: "dia_ly", label: "Địa lí" },
  { key: "gdcd", label: "GDCD" },
  { key: "khxh", label: "KHXH" },
  { key: "tieng_anh", label: "Tiếng Anh" },
  { key: "tieng_phap", label: "Tiếng Pháp" },
  { key: "tieng_nga", label: "Tiếng Nga" },
  { key: "tieng_duc", label: "Tiếng Đức" },
  { key: "tieng_nhat", label: "Tiếng Nhật" },
  { key: "tieng_trung", label: "Tiếng Trung" },
];

/**
 * Non-score columns worth showing in the results table. Only the 2016 dataset
 * populates these; the same all-NULL filter that hides unused subjects hides
 * them elsewhere, so no per-dataset conditional is needed.
 */
export const IDENTITY_COLUMNS = [
  { key: "ten_cum_thi", label: "Cụm thi" },
  { key: "gioi_tinh", label: "GT" },
];

/** True when at least one row carries a value for `key`. */
export function hasAnyValue(rows, key) {
  return rows.some((row) => row[key] !== null && row[key] !== undefined);
}
