#!/usr/bin/env node
/**
 * Build the SQLite database for one or all datasets, verify it, then gzip it.
 *
 * The dataset list comes from src/datasets.js so it is written in exactly one
 * place.
 *
 * Output goes to .build/public/db/ — the directory Vite copies as its publicDir.
 * Only the .gz survives: shipping a 100+ MB uncompressed database is made
 * structurally impossible rather than left to a cleanup step.
 *
 * VERIFICATION IS THE POINT OF THIS SCRIPT, not an extra.
 *
 * Until now nothing between the parser and the public site asserted that a
 * database actually had data in it. The parser logs a file-level failure and
 * continues, returns success regardless, and finishes cleanly even at zero rows;
 * this script gzipped whatever it got; and scripts/assemble-site.js only greps
 * *filenames* for stray .db files. So a reader that silently under-produced
 * would publish a truncated dataset with green CI and no red signal anywhere.
 *
 * The guard below closes that: a build whose row count does not match the known
 * figure, or whose artifact is implausibly small, fails the pipeline.
 *
 * Usage:
 *   node go-parser/scripts/build-db.js            # all four datasets
 *   node go-parser/scripts/build-db.js 2017-old   # just one
 */

import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync, existsSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { DatabaseSync } from "node:sqlite";

import { DATASET_IDS, DATASETS } from "../../src/datasets.js";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const BIN = resolve(ROOT, "go-parser/bin/xlsxread");
const OUT_DIR = resolve(ROOT, ".build/public/db");

/**
 * Known-good row counts, from docs/data-pipeline.md.
 *
 * The inputs are frozen historical exam results, so these are exact, not
 * approximate. A deviation of even one row means something changed that nobody
 * intended — treat it as a build failure, not a warning.
 */
const EXPECTED_ROWS = {
  "2016": 877461,
  "2017": 861068,
  "2017-old": 847348,
  "2017-old2": 679764,
};

/** A gzipped database far below its usual size means a truncated build. */
const MIN_SIZE_RATIO = 0.9;

const requested = process.argv.slice(2);
const unknown = requested.filter((id) => !DATASET_IDS.includes(id));
if (unknown.length) {
  console.error(`unknown dataset(s): ${unknown.join(", ")}`);
  console.error(`known: ${DATASET_IDS.join(", ")}`);
  process.exit(2);
}
const targets = requested.length ? requested : DATASET_IDS;

if (!existsSync(BIN)) {
  console.error(`parser binary not found at ${BIN}`);
  console.error("run: npm run build:go");
  process.exit(1);
}

mkdirSync(OUT_DIR, { recursive: true });

for (const id of targets) {
  const db = resolve(OUT_DIR, `${id}.db`);

  // execFileSync throws on a non-zero exit, so a parser failure aborts the run.
  execFileSync(
    BIN,
    [
      "build",
      "--schema",
      resolve(ROOT, `parser/configs/${id}.yml`),
      "--input",
      resolve(ROOT, `data/${id}`),
      "--output",
      db,
    ],
    { stdio: "inherit" },
  );

  // --- guard: the database must contain what it is supposed to contain ---
  const expected = EXPECTED_ROWS[id];
  if (expected === undefined) {
    console.error(`no expected row count recorded for ${id}; add one to EXPECTED_ROWS`);
    process.exit(1);
  }
  const conn = new DatabaseSync(db, { readOnly: true });
  const actual = conn.prepare("SELECT COUNT(*) c FROM student").get().c;
  conn.close();
  if (actual !== expected) {
    console.error(`\n${id}: row count ${actual}, expected ${expected}`);
    console.error("Refusing to publish — the build did not reproduce the known dataset.");
    process.exit(1);
  }
  console.log(`  ✓ ${id}: ${actual} rows (matches expected)`);

  // -9 without -k: the raw .db must not reach the published artifact.
  rmSync(`${db}.gz`, { force: true });
  execFileSync("gzip", ["-9", db], { stdio: "inherit" });

  const gz = `${db}.gz`;
  const sizeMb = statSync(gz).size / 1024 / 1024;
  const nominal = DATASETS.find((d) => d.id === id)?.dbSizeMb;
  if (nominal && sizeMb < nominal * MIN_SIZE_RATIO) {
    console.error(
      `\n${id}: ${sizeMb.toFixed(1)} MB is below ${(nominal * MIN_SIZE_RATIO).toFixed(1)} MB ` +
        `(${MIN_SIZE_RATIO * 100}% of the expected ${nominal} MB)`,
    );
    console.error("Refusing to publish — the artifact looks truncated.");
    process.exit(1);
  }

  console.log(`  → db/${id}.db.gz (${sizeMb.toFixed(1)} MB)\n`);
}
