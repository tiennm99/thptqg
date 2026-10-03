import { createDbWorker } from "sql.js-httpvfs";
import workerUrl from "sql.js-httpvfs/dist/sqlite.worker.js?url";
import wasmUrl from "sql.js-httpvfs/dist/sql-wasm.wasm?url";
import { probeDatabase } from "./db-probe.js";

/**
 * The database is read where it lies. SQLite asks for pages, the virtual file
 * system turns each into an HTTP range request, and only the pages a query
 * touches ever cross the network — a few hundred KB for a lookup, against the
 * 45 MB the whole file used to cost before the first query.
 *
 * That only holds while every query is index-driven. The schema exists for it:
 * name_word serves name search, and the score indexes serve the SQL presets.
 * An unindexed query walks the table and pulls all 100+ MB of it, which is what
 * the byte budget below is for.
 */

// Must equal the page size the parser writes (PRAGMA page_size in
// parser/internal/writer/writer.go), so one request is exactly one page. A
// mismatch makes every logical page read span two requests.
const CHUNK_BYTES = 1024;

/** Generous for indexed work: a name search costs well under 1 MB. */
export const SEARCH_BUDGET_BYTES = 25 * 1024 * 1024;

/** What the SQL tab gets once the user has accepted the cost of a scan. */
export const PLAYGROUND_BUDGET_BYTES = 250 * 1024 * 1024;

/**
 * What one query cost over the network: `{ requests, bytes, ms }`.
 */

/**
 * One remotely-paged database, with the load state the UI needs.
 *
 * `budgetBytes` is a hard ceiling for the worker's lifetime: past it a query
 * fails instead of quietly downloading the file. Raising it means a new worker,
 * which costs only the header pages.
 */
export class RemoteDatabase {
  ready = $state(false);
  error = $state(null);
  /** Bytes fetched by this database so far, prefetch included. */
  bytesRead = $state(0);
  /** HTTP range requests issued so far. */
  requests = $state(0);
  /** What the most recent query cost on its own. */
  lastCost = $state(null);

  #worker = null;
  #opening;
  #closed = false;
  // Previous cumulative reading, so a query's own cost is a subtraction.
  #seen = { requests: 0, bytes: 0 };

  /**
   * `source` carries both forms of the same location: `url` is the file, and
   * `urlPrefix` is what the library appends the chunk index to. They must
   * agree — see dbOf/dbPrefixOf, which derive one from the other.
   */
  constructor(source, budgetBytes = SEARCH_BUDGET_BYTES) {
    this.url = source.url;
    this.urlPrefix = source.urlPrefix;
    this.budgetBytes = budgetBytes;
    this.#opening = this.#open();
  }

  async #open() {
    const opened = performance.now();
    try {
      // Chunked mode over a single chunk, which looks odd but is the only way
      // to tell this library how long the file is: the worker reads
      // databaseLengthBytes in chunked mode and hardcodes the length to
      // undefined in full mode. Left to itself it sizes the file with a HEAD
      // request, and GitHub Pages answers that with the gzipped length, which
      // it then refuses to use.
      //
      // One chunk covers the whole database, so the chunk index is always 0
      // and every request goes to urlPrefix + "0" — the file the assembler
      // publishes as <id>.sqlite30.
      const databaseLengthBytes = await probeDatabase(this.url, CHUNK_BYTES);
      const worker = await createDbWorker(
        [
          {
            from: "inline",
            config: {
              serverMode: "chunked",
              urlPrefix: this.urlPrefix,
              serverChunkSize: databaseLengthBytes,
              databaseLengthBytes,
              suffixLength: 1,
              requestChunkSize: CHUNK_BYTES,
            },
          },
        ],
        workerUrl,
        wasmUrl,
        this.budgetBytes,
      );
      if (this.#closed) throw new Error("closed");
      this.#worker = worker;
      this.ready = true;
      // Opening is not free either: the header and schema pages are read before
      // any query runs, and that shows up in every later session total.
      await this.#account(worker, `open ${this.url}`, performance.now() - opened);
      return worker;
    } catch (err) {
      if (!this.#closed) {
        this.error = message(err);
        this.ready = false;
      }
      throw err;
    }
  }

  /**
   * Run a query and return its rows as objects.
   *
   * `label` names the query in the console trace — the only way to see what a
   * search actually costs, since the byte count depends on how much the read
   * heads prefetched, not just on the pages the plan needed.
   */
  async query(sql, params = [], label) {
    const worker = this.#worker ?? (await this.#opening);
    const started = performance.now();
    try {
      return await worker.db.query(sql, ...params);
    } finally {
      await this.#account(worker, label ?? firstLine(sql), performance.now() - started);
    }
  }

  /**
   * Read the cumulative counters and report the delta.
   *
   * getStats() rather than the worker's `bytesRead`: that one is the budget
   * counter and resets itself to zero when a query trips the ceiling, so it
   * would under-report exactly when the number matters most.
   */
  async #account(worker, label, ms) {
    const stats = await worker.worker.getStats().catch(() => null);
    if (!stats) return;

    const cost = {
      requests: stats.totalRequests - this.#seen.requests,
      bytes: stats.totalFetchedBytes - this.#seen.bytes,
      ms,
    };
    this.#seen = { requests: stats.totalRequests, bytes: stats.totalFetchedBytes };
    this.requests = stats.totalRequests;
    this.bytesRead = stats.totalFetchedBytes;
    this.lastCost = cost;

    console.info(
      `[httpvfs] ${label} — ${cost.requests} request(s), ${formatBytes(cost.bytes)}, ${ms.toFixed(0)} ms` +
        ` · session ${stats.totalRequests} request(s), ${formatBytes(stats.totalFetchedBytes)}` +
        ` of ${formatBytes(stats.totalBytes)}`,
    );
  }

  /**
   * Drop this database. createDbWorker owns the Worker and exposes no handle to
   * it, so the thread outlives this call; a page creates at most one per
   * dataset and one more if the SQL budget is raised, which is why that is
   * tolerable rather than a leak worth working around.
   */
  close() {
    this.#closed = true;
    this.#worker = null;
    this.ready = false;
  }
}

/** True when a query failed because it would have exceeded the byte budget. */
export function isBudgetError(err) {
  return /maxBytesToRead|too much data|exceeded/i.test(message(err));
}

function message(err) {
  return err instanceof Error ? err.message : String(err);
}

/** Enough of a query to recognise it in the console. */
function firstLine(sql) {
  const line = sql.trim().split("\n")[0];
  return line.length > 70 ? `${line.slice(0, 70)}…` : line;
}

export function formatBytes(n) {
  return n < 1024 * 1024 ? `${Math.round(n / 1024)} KB` : `${(n / 1048576).toFixed(1)} MB`;
}
