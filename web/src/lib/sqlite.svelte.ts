import initSqlJs, { type Database } from "sql.js";

// The sql.js engine itself, fetched from the upstream CDN rather than bundled.
// It is a hard runtime dependency: if this URL is unreachable, initSqlJs()
// rejects and no dataset can be opened at all, whatever the database fetch does.
const SQL_WASM_URL = "https://sql.js.org/dist/sql-wasm.wasm";

/**
 * A SQLite database loaded from a URL into sql.js, with reactive load state.
 *
 * The whole file is downloaded and decompressed before the first query: a `.gz`
 * URL is inflated in the browser. Nothing streams — sql.js needs the complete
 * image in memory.
 */
export class SqliteSource {
  db = $state<Database | null>(null);
  loading = $state(true);
  error = $state<string | null>(null);
  progress = $state(0);

  #cancelled = false;

  constructor(url: string) {
    void this.#load(url);
  }

  async #load(url: string) {
    try {
      const SQL = await initSqlJs({ locateFile: () => SQL_WASM_URL });

      const response = await fetch(url);
      if (!response.ok) throw new Error(`Failed to fetch database: ${response.status}`);

      const contentLength = Number(response.headers.get("Content-Length")) || 0;
      const reader = response.body!.getReader();
      const chunks: Uint8Array[] = [];
      let received = 0;

      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
        received += value.length;
        if (contentLength > 0) {
          this.progress = Math.round((received / contentLength) * 100);
        }
      }
      if (this.#cancelled) return;

      const blob = new Blob(chunks as BlobPart[]);
      const bytes = url.endsWith(".gz")
        ? await new Response(blob.stream().pipeThrough(new DecompressionStream("gzip"))).arrayBuffer()
        : await blob.arrayBuffer();
      if (this.#cancelled) return;

      this.db = new SQL.Database(new Uint8Array(bytes));
      this.loading = false;
    } catch (err) {
      if (this.#cancelled) return;
      this.error = err instanceof Error ? err.message : String(err);
      this.loading = false;
    }
  }

  /** Release the database. Call from the owning component's teardown. */
  close() {
    this.#cancelled = true;
    this.db?.close();
    this.db = null;
  }
}

/** Run a statement and return its rows as objects. */
export function queryRows<T>(db: Database, sql: string, params?: Record<string, unknown>): T[] {
  const stmt = db.prepare(sql);
  try {
    if (params) stmt.bind(params as never);
    const rows: T[] = [];
    while (stmt.step()) rows.push(stmt.getAsObject() as T);
    return rows;
  } finally {
    stmt.free();
  }
}
