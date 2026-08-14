/**
 * How long the database is, asked in the one way that survives a CDN.
 *
 * `sql.js-httpvfs` sizes a file with a HEAD request. That request carries no
 * Range header, so the browser advertises gzip, and GitHub Pages answers with
 * `Content-Encoding: gzip` and the length of the *compressed* body — 64 MB for
 * a 288 MB database. The library rightly refuses to believe it and gives up
 * with "Length of the file not known. It must either be supplied in the config
 * or given by the HTTP server."
 *
 * Range requests do not have that problem: the Fetch standard requires
 * `Accept-Encoding: identity` on any request carrying a Range header, so the
 * page reads that do the actual work always come back uncompressed. Asking for
 * the first hundred bytes therefore yields both a trustworthy total, from
 * Content-Range, and the file header itself to check it against.
 */

/** A SQLite file opens with these characters and then a NUL byte. */
const MAGIC = "SQLite format 3";

/** Enough for the whole SQLite header. */
const PROBE_BYTES = 100;

/** Page size lives at offset 16, big-endian; the value 1 encodes 65536. */
const PAGE_SIZE_OFFSET = 16;

/** Total size of the representation, from `bytes <from>-<to>/<total>`. */
export function parseTotalBytes(contentRange) {
  const match = /\/\s*(\d+)\s*$/.exec(contentRange ?? "");
  if (!match) {
    throw new Error(
      `the server did not say how large the database is (Content-Range: ${contentRange ?? "absent"})`,
    );
  }
  return Number(match[1]);
}

/** Page size the file was written with, from its header. */
export function readPageSize(header) {
  const raw = (header[PAGE_SIZE_OFFSET] << 8) | header[PAGE_SIZE_OFFSET + 1];
  return raw === 1 ? 65536 : raw;
}

/** True when these bytes begin a SQLite database. */
export function looksLikeSqlite(header) {
  const text = Array.from(MAGIC).every((ch, i) => header[i] === ch.charCodeAt(0));
  return text && header[MAGIC.length] === 0;
}

/**
 * Read the file header over a range request and return the database's length.
 *
 * Doubles as the check that the host is serving raw database bytes: a body that
 * does not start with the SQLite magic means something rewrote it in transit —
 * compression being the way that happens — and every later page read would be
 * reading the wrong bytes.
 */
export async function probeDatabase(url, expectedPageSize, fetchImpl = fetch) {
  const response = await fetchImpl(url, { headers: { Range: `bytes=0-${PROBE_BYTES - 1}` } });

  if (response.status !== 206) {
    throw new Error(
      `${url}: expected 206 for a range request, got ${response.status}. ` +
        "The host must serve byte ranges of the database.",
    );
  }

  const total = parseTotalBytes(response.headers.get("Content-Range"));
  const header = new Uint8Array(await response.arrayBuffer());

  if (!looksLikeSqlite(header)) {
    throw new Error(
      `${url}: the first bytes are not a SQLite header, so the host is not ` +
        "serving the database as stored — check for Content-Encoding on ranged responses.",
    );
  }

  const pageSize = readPageSize(header);
  if (expectedPageSize && pageSize !== expectedPageSize) {
    // Not fatal: it still reads correctly, just at more requests per page.
    console.warn(
      `[httpvfs] ${url} has page size ${pageSize}, but requests are ${expectedPageSize} bytes. ` +
        "Every page read now spans more than one request.",
    );
  }

  return total;
}
