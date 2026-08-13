#!/usr/bin/env node
/**
 * Build the SQLite database for one or all datasets, then gzip it.
 *
 * Replaces the six per-dataset npm scripts the two old projects carried. The
 * dataset list comes from src/datasets.js so it is written in exactly one place.
 *
 * Output goes to .build/public/db/ — the directory Vite copies as its publicDir.
 * Only the .gz survives: shipping a 100+ MB uncompressed database is made
 * structurally impossible rather than left to a cleanup step.
 *
 * Usage:
 *   node parser/scripts/build-db.js            # all four datasets
 *   node parser/scripts/build-db.js 2017-old   # just one
 */

import { execFileSync } from "node:child_process";
import { mkdirSync, rmSync, existsSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { DATASET_IDS } from "../../src/datasets.js";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const BIN = resolve(ROOT, "parser/target/release/xlsxread");
const OUT_DIR = resolve(ROOT, ".build/public/db");

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
  console.error("run: npm run build:rust");
  process.exit(1);
}

mkdirSync(OUT_DIR, { recursive: true });

for (const id of targets) {
  const db = resolve(OUT_DIR, `${id}.db`);

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

  // -9 without -k: the raw .db must not reach the published artifact.
  rmSync(`${db}.gz`, { force: true });
  execFileSync("gzip", ["-9", db], { stdio: "inherit" });
  console.log(`  → db/${id}.db.gz\n`);
}
