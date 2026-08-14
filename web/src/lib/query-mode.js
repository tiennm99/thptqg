/**
 * Classify a search box query as an exam ID (số báo danh) or a name. Shared by
 * App.jsx and search-form.jsx so the two cannot disagree on what a query is.
 *
 * SBD formats across both exam years:
 *
 *   49008235     2017 — 8 digits, first two identify the province
 *   017006021    2016 — 9 digits including a leading zero
 *   BAL000001    2016 — 2-4 letter exam-cluster code then digits
 *
 * The letter prefix is not an edge case: 70% of the 2016 candidates have one,
 * so a digits-only pattern would break exam-ID lookup for most of that year.
 */

const SBD_PATTERN = /^[A-Za-z]{0,4}\d+$/;

export const MIN_SBD_DIGITS = 3;
export const MIN_NAME_CHARS = 2;

/** True when the query looks like an exam ID rather than a name. */
export function isExamId(query) {
  return SBD_PATTERN.test(query.trim());
}

/**
 * Normalise an exam ID for lookup. Letter prefixes are stored upper-case, so a
 * user typing "bal000001" still matches. No-op for all-digit IDs.
 */
export function normaliseExamId(query) {
  return query.trim().toUpperCase();
}

function digitCount(str) {
  return (str.match(/\d/g) ?? []).length;
}

/**
 * Decide what the user is searching for, and what hint to show beneath the
 * field. Returns `{ mode, hint }` where mode is one of:
 * empty | sbd | sbd-short | name | name-short.
 */
export function detectMode(raw) {
  const q = raw.trim();
  if (!q) return { mode: "empty", hint: "Gõ SBD (số báo danh) hoặc họ tên để tìm" };

  if (isExamId(q)) {
    return digitCount(q) >= MIN_SBD_DIGITS
      ? { mode: "sbd", hint: "Tìm theo số báo danh · khớp chính xác" }
      : { mode: "sbd-short", hint: `Cần ít nhất ${MIN_SBD_DIGITS} chữ số` };
  }

  return q.length >= MIN_NAME_CHARS
    ? { mode: "name", hint: "Tìm theo họ tên · không phân biệt dấu và hoa/thường" }
    : { mode: "name-short", hint: `Cần ít nhất ${MIN_NAME_CHARS} ký tự` };
}
