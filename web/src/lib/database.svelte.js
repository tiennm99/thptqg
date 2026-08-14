import { keepOnly, matchAny, matchVersion, openCache } from "./db-cache.js";
import initSqlJs from "sql.js";
import wasmUrl from "sql.js/dist/sql-wasm.wasm?url";

/**
 * The database is downloaded once and then queried in memory.
 *
 * The previous design read it where it lay, a page at a time over HTTP range
 * requests. That works, but every page read is a separate round trip and they
 * are serial, so a search that touched 390 pages waited 17 seconds for 608 KB
 * — and the first visitor after a deploy waited another 20 for the CDN to fill
 * its cache with a 288 MB object. Downloading trades one large transfer, which
 * a browser and a CDN are both good at, for queries that cost nothing.
 *
 * The cost of that trade is memory: SQLite runs inside WebAssembly, so the
 * whole file sits in the tab's address space for as long as the page is open.
 * That is what the gate on the dataset page tells the visitor before they
 * commit to it.
 *
 * The transfer is paid once per device rather than once per visit: the
 * response is kept in Cache Storage, versioned by ETag so that a redeploy
 * replaces it instead of being served stale. See db-cache.js.
 */

/** Rows to return before a query is considered too broad to render. */
export const MAX_ROWS = 1000;

let sqlPromise = null;

/** The SQLite WASM module, initialised once per tab. */
function sqlEngine() {
  sqlPromise ??= initSqlJs({ locateFile: () => wasmUrl });
  return sqlPromise;
}

/**
 * One dataset's database: its download, and the queries that follow.
 *
 * `expectedBytes` is the uncompressed size from the registry. The server sends
 * the file gzipped, so `Content-Length` counts compressed bytes while the
 * stream yields decompressed ones; progress is measured against the registry
 * figure instead, and the transfer size is reported separately by `weigh()`.
 */
export class LocalDatabase {
  /** Ready to answer queries. */
  ready = $state(false);
  /** Downloading, or opening the file once it has arrived. */
  loading = $state(false);
  error = $state(null);
  /** Uncompressed bytes received so far. */
  received = $state(0);
  /** How long the download and open took, once done. */
  ms = $state(0);
  /** True once the cache has been consulted, so the gate knows what to ask for. */
  checked = $state(false);
  /** True when this copy came off the disk rather than the network. */
  fromCache = $state(false);
  /** Compressed size of the download, as the server reports it. */
  transferBytes = $state(null);

  #db = null;
  #etag = null;

  constructor(url, expectedBytes) {
    this.url = url;
    this.expectedBytes = expectedBytes;
  }

  /**
   * Ask the server what it holds, and the disk what we kept.
   *
   * A copy already on the device costs no network and needs no consent, so it
   * opens straight away. Otherwise this only works out what a download would
   * cost, and the gate waits for the visitor to accept it.
   */
  async prepare() {
    const head = await describe(this.url);
    this.transferBytes = head?.bytes ?? null;
    this.#etag = head?.etag ?? null;

    const cache = await openCache();
    // Without an ETag the server is unreachable; any stored version beats an
    // error page, since these are frozen exam results.
    const stored = this.#etag
      ? await matchVersion(cache, this.url, this.#etag)
      : await matchAny(cache, this.url);

    this.checked = true;
    if (stored) {
      this.fromCache = true;
      await this.open(stored);
    }
  }

  /** Fraction downloaded, 0 to 1, for the progress bar. */
  get progress() {
    if (!this.expectedBytes) return 0;
    return Math.min(1, this.received / this.expectedBytes);
  }

  /**
   * Fetch the database and open it.
   *
   * Read as a stream rather than with response.arrayBuffer() so the visitor
   * sees movement: this is tens of megabytes on a phone connection, and a
   * progress bar is the difference between waiting and giving up.
   */
  async open(stored = null) {
    if (this.ready || this.loading) return;
    this.loading = true;
    this.error = null;
    this.received = 0;
    const started = performance.now();

    try {
      const [SQL, response] = await Promise.all([
        sqlEngine(),
        stored ?? fetchAndCache(this.url, this.#etag),
      ]);
      if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);

      const reader = response.body.getReader();
      const parts = [];
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        parts.push(value);
        this.received += value.length;
      }

      const bytes = new Uint8Array(this.received);
      let at = 0;
      for (const part of parts) {
        bytes.set(part, at);
        at += part.length;
      }

      this.#db = new SQL.Database(bytes);
      this.ready = true;
      this.ms = performance.now() - started;
    } catch (err) {
      this.error = err instanceof Error ? err.message : String(err);
    } finally {
      this.loading = false;
    }
  }

  /**
   * Run a query and return its rows as objects.
   *
   * Synchronous: the database is local, and SQLite answers from memory. A
   * scan of every row takes a few hundred milliseconds, which is why the
   * schema carries no secondary indexes.
   */
  query(sql, params = []) {
    if (!this.#db) throw new Error("Cơ sở dữ liệu chưa sẵn sàng");

    const statement = this.#db.prepare(sql);
    try {
      if (params.length > 0) statement.bind(params);
      const rows = [];
      while (statement.step() && rows.length < MAX_ROWS) rows.push(statement.getAsObject());
      return rows;
    } finally {
      statement.free();
    }
  }

  close() {
    this.#db?.close();
    this.#db = null;
    this.ready = false;
  }
}

/**
 * What the file weighs on the wire, and which version it is.
 *
 * The browser advertises gzip and GitHub Pages compresses this file, so the
 * length is the compressed one — the number that matters to someone on mobile
 * data. The ETag is what keeps a stored copy honest across deploys. Returns
 * null when the server cannot be reached, which is survivable: a stored copy
 * still opens.
 */
async function describe(url) {
  try {
    const response = await fetch(url, { method: "HEAD" });
    if (!response.ok) return null;
    const length = Number(response.headers.get("Content-Length"));
    return {
      bytes: Number.isFinite(length) && length > 0 ? length : null,
      etag: response.headers.get("ETag"),
    };
  } catch {
    return null;
  }
}

/**
 * Fetch the database, handing a second copy of the response to the cache.
 *
 * The clone is what gets stored, so the bytes stream to disk while the caller
 * reads the original for the progress bar. Buffering the whole file a second
 * time here would double the peak memory of the one thing already large enough
 * to matter.
 */
async function fetchAndCache(url, etag) {
  const response = await fetch(url);
  if (response.ok && etag) {
    const copy = response.clone();
    void openCache().then((cache) => keepOnly(cache, url, etag, copy));
  }
  return response;
}

export function formatBytes(n) {
  return n < 1024 * 1024 ? `${Math.round(n / 1024)} KB` : `${(n / 1048576).toFixed(0)} MB`;
}
