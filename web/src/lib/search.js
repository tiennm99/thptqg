import { normaliseExamId } from "./query-mode";
import { toAscii } from "./to-ascii";

export const MAX_RESULTS = 100;

/**
 * Name search over the downloaded database.
 *
 * Each word of the query must appear at the start of a word in the candidate's
 * ASCII name, in any order, so "buu loc" finds "Nguyễn Bửu Lộc". Prefixing the
 * stored name with a space lets one pattern match the first word too.
 *
 * This scans every row. That is the deliberate trade of downloading the file:
 * a scan of 877,460 rows in memory takes a few hundred milliseconds, and
 * paying for it means the published database carries no name index — which was
 * more than half its size.
 */
export function searchByName(db, query) {
  const words = tokenise(query);
  if (words.length === 0) return [];

  const filters = words.map(() => `(' ' || ho_ten_ascii) LIKE ? ESCAPE '\\'`).join(" AND ");
  return db.query(
    `SELECT * FROM student WHERE ${filters} LIMIT ${MAX_RESULTS}`,
    words.map((w) => `% ${escapeLike(w)}%`),
  );
}

/** Exact lookup by exam number, straight down the primary key. */
export function lookupExamId(db, id) {
  return db.query(`SELECT * FROM student WHERE so_bao_danh = ? LIMIT ${MAX_RESULTS}`, [
    normaliseExamId(id),
  ]);
}

/** Fold to ASCII and split into words, the same shape ho_ten_ascii is stored in. */
export function tokenise(query) {
  return toAscii(query).split(/\s+/).filter(Boolean);
}

/** Escape the LIKE wildcards so a user typing % or _ searches for them. */
function escapeLike(s) {
  return s.replace(/[\\%_]/g, (c) => `\\${c}`);
}
