/**
 * The 16 subject columns of the canonical schema, in display order.
 *
 * Mirrors `SCORE_FIELDS` in parser/src/schema.rs. Previously this list was
 * maintained separately in score-table.jsx and student-detail.jsx, which is how
 * they drifted out of sync with each other and with the database.
 *
 * No subject is dataset-specific here. Columns a given exam year has no data
 * for are simply NULL for every row, and the UI hides all-NULL columns — so one
 * list serves 2016 and 2017 without branching.
 */

export const SUBJECTS = [
  { key: "toan", label: "Toán", short: "Toán" },
  { key: "ngu_van", label: "Ngữ văn", short: "Văn" },
  { key: "vat_ly", label: "Vật lí", short: "Lý" },
  { key: "hoa_hoc", label: "Hóa học", short: "Hóa" },
  { key: "sinh_hoc", label: "Sinh học", short: "Sinh" },
  { key: "khtn", label: "KHTN", short: "KHTN" },
  { key: "lich_su", label: "Lịch sử", short: "Sử" },
  { key: "dia_ly", label: "Địa lí", short: "Địa" },
  { key: "gdcd", label: "GDCD", short: "GDCD" },
  { key: "khxh", label: "KHXH", short: "KHXH" },
  { key: "tieng_anh", label: "Tiếng Anh", short: "T.Anh" },
  { key: "tieng_phap", label: "Tiếng Pháp", short: "T.Pháp" },
  { key: "tieng_nga", label: "Tiếng Nga", short: "T.Nga" },
  { key: "tieng_duc", label: "Tiếng Đức", short: "T.Đức" },
  { key: "tieng_nhat", label: "Tiếng Nhật", short: "T.Nhật" },
  { key: "tieng_trung", label: "Tiếng Trung", short: "T.Trung" },
];

/**
 * Non-score columns worth showing in the results table.
 *
 * Only the 2016 dataset populates these; they are NULL throughout the 2017
 * datasets and get hidden by the same all-NULL filter that hides unused
 * subjects, so no per-dataset conditional is needed.
 */
export const IDENTITY_COLUMNS = [
  { key: "ten_cum_thi", label: "Cụm thi" },
  { key: "gioi_tinh", label: "GT" },
];

export const SUBJECT_LABELS = Object.fromEntries(
  SUBJECTS.map((s) => [s.key, s.label]),
);

/** True when at least one row carries a value for `key`. */
export function hasAnyValue(rows, key) {
  return rows.some((row) => row[key] !== null && row[key] !== undefined);
}
