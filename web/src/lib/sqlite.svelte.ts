import { createDbWorker, type SqliteStats, type WorkerHttpvfs } from "sql.js-httpvfs";
import workerUrl from "sql.js-httpvfs/dist/sqlite.worker.js?url";
import wasmUrl from "sql.js-httpvfs/dist/sql-wasm.wasm?url";

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

/** What one query cost over the network. */
export type QueryCost = { requests: number; bytes: number; ms: number };

/**
 * One remotely-paged database, with the load state the UI needs.
 *
 * `budgetBytes` is a hard ceiling for the worker's lifetime: past it a query
 * fails instead of quietly downloading the file. Raising it means a new worker,
 * which costs only the header pages.
 */
export class RemoteDatabase {
  ready = $state(false);
  error = $state<string | null>(null);
  /** Bytes fetched by this database so far, prefetch included. */
  bytesRead = $state(0);
  /** HTTP range requests issued so far. */
  requests = $state(0);
  /** What the most recent query cost on its own. */
  lastCost = $state<QueryCost | null>(null);

  #worker: WorkerHttpvfs | null = null;
  #opening: Promise<WorkerHttpvfs>;
  #closed = false;
  // Previous cumulative reading, so a query's own cost is a subtraction.
  #seen = { requests: 0, bytes: 0 };

  constructor(
    readonly url: string,
    readonly budgetBytes: number = SEARCH_BUDGET_BYTES,
  ) {
    this.#opening = this.#open();
  }

  async #open(): Promise<WorkerHttpvfs> {
    const opened = performance.now();
    try {
      const worker = await createDbWorker(
        [{ from: "inline", config: { serverMode: "full", url: this.url, requestChunkSize: CHUNK_BYTES } }],
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
  async query<T>(sql: string, params: unknown[] = [], label?: string): Promise<T[]> {
    const worker = this.#worker ?? (await this.#opening);
    // Comlink erases the generic when it proxies the method across the worker
    // boundary, so the row type is asserted here rather than inferred.
    const run = worker.db.query as unknown as (sql: string, ...params: unknown[]) => Promise<T[]>;
    const started = performance.now();
    try {
      return await run(sql, ...params);
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
  async #account(worker: WorkerHttpvfs, label: string, ms: number) {
    const read = worker.worker.getStats as unknown as () => Promise<SqliteStats | null>;
    const stats = await read().catch(() => null);
    if (!stats) return;

    const cost: QueryCost = {
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
export function isBudgetError(err: unknown): boolean {
  return /maxBytesToRead|too much data|exceeded/i.test(message(err));
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/** Enough of a query to recognise it in the console. */
function firstLine(sql: string): string {
  const line = sql.trim().split("\n")[0];
  return line.length > 70 ? `${line.slice(0, 70)}…` : line;
}

export function formatBytes(n: number): string {
  return n < 1024 * 1024 ? `${Math.round(n / 1024)} KB` : `${(n / 1048576).toFixed(1)} MB`;
}
