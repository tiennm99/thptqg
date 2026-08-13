#!/usr/bin/env node
/**
 * Compare rebuilt databases against the pre-refactor parity baseline.
 *
 * The schema deliberately changed shape, so a whole-file hash is meaningless.
 * What must hold instead:
 *
 *   1. Row count per dataset is unchanged.
 *   2. Every column that existed before has the same non-NULL count.
 *   3. Every column newly added to a dataset has a non-NULL count of exactly 0.
 *      This is the check that catches the union-regex risk — if the 16-pattern
 *      map starts matching text the narrower per-year map ignored, it shows up
 *      here rather than silently corrupting the dataset.
 *   4. A deterministic sample of students is identical field by field.
 *
 * Exits non-zero on any mismatch.
 *
 * Usage:
 *   node verify-parity.js <baseline.json> <current.json>
 */

import { readFileSync } from "node:fs";

/**
 * Foreign-language scores the pre-refactor configs silently discarded.
 *
 * The 2016 config listed 12 subject regexes and the 2017 configs listed 14;
 * neither list was complete. Candidates could sit German, Japanese and Russian
 * in both exam years, so every one of these students previously ended up with
 * no foreign-language score at all.
 *
 * Unifying to the canonical 16 patterns recovers them. Verified real, not
 * spurious matches: across all four datasets every student holds either zero
 * or exactly one foreign language — never two — and each affected student had
 * all language columns NULL beforehand.
 *
 * These exact counts are approved. Any other newly-populated column, or any
 * drift in these numbers, still fails the gate.
 */
const APPROVED_RECOVERY = {
  "2016": { tieng_nga: 182 },
  "2017": { tieng_duc: 93, tieng_nhat: 512 },
  "2017-old": { tieng_duc: 85, tieng_nhat: 484 },
  "2017-old2": { tieng_duc: 22, tieng_nhat: 313 },
};

const [baselinePath, currentPath] = process.argv.slice(2);
if (!baselinePath || !currentPath) {
  console.error("usage: verify-parity.js <baseline.json> <current.json>");
  process.exit(2);
}

const baseline = JSON.parse(readFileSync(baselinePath, "utf8"));
const current = JSON.parse(readFileSync(currentPath, "utf8"));

const failures = [];
const notes = [];

for (const dataset of Object.keys(baseline)) {
  const b = baseline[dataset];
  const c = current[dataset];

  if (!c) {
    failures.push(`${dataset}: missing from current stats`);
    continue;
  }

  // 1. Row count
  if (b.rowCount !== c.rowCount) {
    failures.push(
      `${dataset}: row count ${b.rowCount} → ${c.rowCount} (${c.rowCount - b.rowCount >= 0 ? "+" : ""}${c.rowCount - b.rowCount})`,
    );
  }

  // 2. Pre-existing columns keep their non-NULL counts
  for (const col of b.columns) {
    if (!(col in c.nonNull)) {
      failures.push(`${dataset}.${col}: column disappeared from schema`);
      continue;
    }
    if (b.nonNull[col] !== c.nonNull[col]) {
      failures.push(
        `${dataset}.${col}: non-NULL ${b.nonNull[col]} → ${c.nonNull[col]} (${c.nonNull[col] - b.nonNull[col] >= 0 ? "+" : ""}${c.nonNull[col] - b.nonNull[col]})`,
      );
    }
  }

  // 3. Newly added columns must be NULL — except the approved recoveries,
  //    which must match their approved count exactly.
  const approved = APPROVED_RECOVERY[dataset] ?? {};
  const added = c.columns.filter((col) => !b.columns.includes(col));
  const recovered = [];
  for (const col of added) {
    const expected = approved[col] ?? 0;
    if (c.nonNull[col] !== expected) {
      failures.push(
        expected === 0
          ? `${dataset}.${col}: new column has ${c.nonNull[col]} non-NULL values, expected 0`
          : `${dataset}.${col}: recovered ${c.nonNull[col]} values, approved count is ${expected}`,
      );
    } else if (expected > 0) {
      recovered.push(`${col}=${expected}`);
    }
  }
  // An approved recovery that vanished means the union patterns regressed.
  for (const [col, expected] of Object.entries(approved)) {
    if (!added.includes(col)) {
      failures.push(
        `${dataset}.${col}: expected ${expected} recovered values but column is not new`,
      );
    }
  }
  if (added.length) {
    const nulls = added.length - recovered.length;
    notes.push(
      `${dataset}: +${added.length} new columns (${nulls} all-NULL as expected` +
        (recovered.length ? `, recovered ${recovered.join(", ")}` : "") +
        ")",
    );
  }

  // 4. Deterministic sample compared field by field
  if (b.sampleCount !== c.sampleCount) {
    failures.push(
      `${dataset}: sample size ${b.sampleCount} → ${c.sampleCount}`,
    );
  }
  for (const [sbd, bRow] of Object.entries(b.sample)) {
    const cRow = c.sample[sbd];
    if (!cRow) {
      failures.push(`${dataset}: sampled student ${sbd} missing after rebuild`);
      continue;
    }
    for (const [field, bVal] of Object.entries(bRow)) {
      if (cRow[field] !== bVal) {
        failures.push(
          `${dataset}: student ${sbd} field ${field}: ${JSON.stringify(bVal)} → ${JSON.stringify(cRow[field])}`,
        );
      }
    }
  }

  const sizeDelta = ((c.sizeBytes - b.sizeBytes) / b.sizeBytes) * 100;
  notes.push(
    `${dataset}: ${c.rowCount} rows, ${b.columns.length} → ${c.columns.length} cols, size ${sizeDelta >= 0 ? "+" : ""}${sizeDelta.toFixed(1)}%`,
  );
}

for (const n of notes) console.log(`  ${n}`);

if (failures.length) {
  console.error(`\nPARITY FAILED — ${failures.length} mismatch(es):\n`);
  for (const f of failures) console.error(`  ✗ ${f}`);
  process.exit(1);
}

console.log("\nPARITY OK — row counts, per-column non-NULL counts and samples all match.");
