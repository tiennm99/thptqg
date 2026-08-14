import { normaliseExamId } from "./query-mode";
import type { RemoteDatabase } from "./sqlite.svelte";
import { toAscii } from "./to-ascii";
import type { Student } from "./types";

export const MAX_RESULTS = 100;

/**
 * Name search over the name_word table.
 *
 * A query is matched word by word, each word as a prefix, in any order — so
 * "buu loc" finds "Nguyễn Bửu Lộc". The work is arranged so that only one word
 * is ever seeked on and the rest are filtered inside the same b-tree:
 *
 *  1. ask name_word_freq how many entries each word prefix covers (the whole
 *     vocabulary is ~4,400 rows, so this is a couple of pages);
 *  2. seek on the rarest one — for a real name that is a few hundred to a few
 *     thousand entries rather than the 300,000 a word like "thi" would walk;
 *  3. filter the other words against the ho_ten_ascii copy carried in
 *     name_word, so nothing is read from student until a row has matched;
 *  4. join to student for the rows that survive, at most MAX_RESULTS of them.
 *
 * Every step is an index seek. A search costs a few hundred KB.
 */
export async function searchByName(db: RemoteDatabase, query: string): Promise<Student[]> {
  const words = tokenise(query);
  if (words.length === 0) return [];

  const seek = await rarest(db, words);
  const others = words.filter((w) => w !== seek);

  // A word matches at a word boundary: the leading space makes the first word
  // reachable by the same pattern as the rest.
  const filters = others.map(() => `(' ' || w.ho_ten_ascii) LIKE ? ESCAPE '\\'`);
  const sql = `
    SELECT s.* FROM name_word w
      JOIN student s ON s.so_bao_danh = w.so_bao_danh
    WHERE w.word >= ? AND w.word < ?${filters.length ? " AND " + filters.join(" AND ") : ""}
    LIMIT ${MAX_RESULTS}`;

  return db.query<Student>(sql, [seek, upperBound(seek), ...others.map((w) => `% ${escapeLike(w)}%`)]);
}

/** Exact lookup by exam number: a primary-key seek, a few pages. */
export async function lookupExamId(db: RemoteDatabase, id: string): Promise<Student[]> {
  return db.query<Student>("SELECT * FROM student WHERE so_bao_danh = ? LIMIT ?", [
    normaliseExamId(id),
    MAX_RESULTS,
  ]);
}

/** Fold to ASCII and split into words, the same shape name_word was built in. */
export function tokenise(query: string): string[] {
  return toAscii(query).split(/\s+/).filter(Boolean);
}

/**
 * The word whose prefix covers the fewest entries, which is the one worth
 * seeking on. One round trip for all of them.
 */
async function rarest(db: RemoteDatabase, words: string[]): Promise<string> {
  if (words.length === 1) return words[0];

  const sql = words
    .map(() => "SELECT ? AS word, COALESCE(SUM(n), 0) AS n FROM name_word_freq WHERE word >= ? AND word < ?")
    .join(" UNION ALL ");
  const params = words.flatMap((w) => [w, w, upperBound(w)]);

  const counts = await db.query<{ word: string; n: number }>(sql, params);
  let best = words[0];
  let bestN = Infinity;
  for (const { word, n } of counts) {
    if (n < bestN) {
      best = word;
      bestN = n;
    }
  }
  return best;
}

/**
 * The exclusive upper bound of a prefix range. U+FFFF sorts above any character
 * that can follow the prefix, which is what turns "starts with" into a range
 * the index can seek.
 */
function upperBound(prefix: string): string {
  return prefix + "￿";
}

/** Escape the LIKE wildcards so a user typing % or _ searches for them. */
function escapeLike(s: string): string {
  return s.replace(/[\\%_]/g, (c) => `\\${c}`);
}
