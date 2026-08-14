/**
 * Vietnamese → ASCII fold, used for accent-insensitive search.
 *
 * This MUST produce the same string as `ToAscii` in
 * parser/internal/transform/transform.go, which is what wrote the
 * `ho_ten_ascii` column being searched. The four steps are the contract:
 *
 *   1. NFD decompose
 *   2. drop combining marks in U+0300..U+036F — a literal range, not a category
 *   3. đ/Đ → d, which NFD does not decompose
 *   4. lowercase
 *
 * to-ascii.test.ts pins the pairs both sides have to agree on.
 */
export function toAscii(str: string): string {
  return str
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/đ/gi, "d")
    .toLowerCase();
}

/** True when every character is ASCII, i.e. the query carries no diacritics. */
export function isAsciiOnly(str: string): boolean {
  for (let i = 0; i < str.length; i++) {
    if (str.charCodeAt(i) > 127) return false;
  }
  return true;
}
