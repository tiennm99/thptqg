/**
 * The 16 subject columns of the canonical schema, in display order.
 *
 * Mirrors `ScoreFields` in parser/internal/schema/schema.go. Previously this list was
 * maintained separately in score-table.jsx and student-detail.jsx, which is how
 * they drifted out of sync with each other and with the database.
 *
 * No subject is dataset-specific here. Columns a given exam year has no data
 * for are simply NULL for every row, and the UI hides all-NULL columns — so one
 * list serves 2016 and 2017 without branching.
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
 * Non-score columns worth showing in the results table.
 *
 * Only the 2016 dataset populates these; they are NULL throughout 2017
 * and get hidden by the same all-NULL filter that hides unused
 * subjects, so no per-dataset conditional is needed.
 */
export const IDENTITY_COLUMNS = [
  { key: "ten_cum_thi", label: "Cụm thi" },
  { key: "gioi_tinh", label: "GT" },
];

/** True when at least one row carries a value for `key`. */
export function hasAnyValue(rows, key) {
  return rows.some((row) => row[key] !== null && row[key] !== undefined);
}
