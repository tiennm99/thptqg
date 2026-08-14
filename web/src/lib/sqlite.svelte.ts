import { createDbWorker, type WorkerHttpvfs } from "sql.js-httpvfs";
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
  /** Bytes fetched so far, refreshed after every query. */
  bytesRead = $state(0);

  #worker: WorkerHttpvfs | null = null;
  #opening: Promise<WorkerHttpvfs>;
  #closed = false;

  constructor(
    readonly url: string,
    readonly budgetBytes: number = SEARCH_BUDGET_BYTES,
  ) {
    this.#opening = this.#open();
  }

  async #open(): Promise<WorkerHttpvfs> {
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
      return worker;
    } catch (err) {
      if (!this.#closed) {
        this.error = message(err);
        this.ready = false;
      }
      throw err;
    }
  }

  /** Run a query and return its rows as objects. */
  async query<T>(sql: string, params: unknown[] = []): Promise<T[]> {
    const worker = this.#worker ?? (await this.#opening);
    // Comlink erases the generic when it proxies the method across the worker
    // boundary, so the row type is asserted here rather than inferred.
    const run = worker.db.query as unknown as (sql: string, ...params: unknown[]) => Promise<T[]>;
    try {
      return await run(sql, ...params);
    } finally {
      // Comlink proxies the property, so this is a round trip; worth it because
      // the number is the only honest feedback about what a query cost.
      this.bytesRead = await worker.worker.bytesRead;
    }
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
