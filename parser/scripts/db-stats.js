#!/usr/bin/env node
/**
 * Dump per-dataset statistics from built SQLite databases as JSON.
 *
 * Used twice: once against the pre-refactor databases to capture a baseline,
 * and again after the schema unification. Comparing the two outputs is what
 * proves no score data was lost or invented.
 *
 * Schema-agnostic on purpose — columns come from PRAGMA table_info, so the same
 * script runs against the old 18/20-column tables and the new 22-column one.
 *
 * Usage:
 *   node db-stats.js <label>=<db-path> [<label>=<db-path> ...] > stats.json
 */

import { DatabaseSync } from "node:sqlite";
import { statSync } from "node:fs";

// Deterministic value-level sample: every SBD ending in these digits.
// Re-running against a rebuilt DB compares the same students.
const SAMPLE_SUFFIX = "0000";

function collect(dbPath) {
  const db = new DatabaseSync(dbPath, { readOnly: true });

  const columns = db
    .prepare("PRAGMA table_info(student)")
    .all()
    .map((c) => c.name);

  const rowCount = db.prepare("SELECT COUNT(*) AS c FROM student").get().c;

  // One pass over the table counting non-NULLs for every column at once.
  const sums = columns
    .map((c) => `SUM(CASE WHEN "${c}" IS NOT NULL THEN 1 ELSE 0 END) AS "${c}"`)
    .join(", ");
  const nonNull = db.prepare(`SELECT ${sums} FROM student`).get();

  const sample = db
    .prepare(
      `SELECT * FROM student WHERE so_bao_danh LIKE '%${SAMPLE_SUFFIX}'
       ORDER BY so_bao_danh`,
    )
    .all();

  db.close();

  return {
    rowCount,
    columns,
    nonNull: Object.fromEntries(columns.map((c) => [c, Number(nonNull[c])])),
    sizeBytes: statSync(dbPath).size,
    sampleCount: sample.length,
    // Store the sample keyed by SBD so a later diff can report which student
    // and which field changed, not just that something did.
    sample: Object.fromEntries(sample.map((r) => [r.so_bao_danh, r])),
  };
}

const args = process.argv.slice(2);
if (args.length === 0) {
  console.error("usage: db-stats.js <label>=<db-path> [...]");
  process.exit(2);
}

const out = {};
for (const arg of args) {
  const idx = arg.indexOf("=");
  if (idx === -1) {
    console.error(`bad argument (expected label=path): ${arg}`);
    process.exit(2);
  }
  const label = arg.slice(0, idx);
  const path = arg.slice(idx + 1);
  process.stderr.write(`collecting ${label} from ${path}\n`);
  out[label] = collect(path);
}

process.stdout.write(JSON.stringify(out, null, 2) + "\n");
