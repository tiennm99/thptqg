// Compare two builds of the 2017 database, column by column.
//
// BROKEN as committed, for two reasons that both predate the repo unification:
//   - `better-sqlite3` is imported but declared in no package.json
//   - the public/ and backup/ paths it reads no longer exist
//
// Left unchanged rather than half-fixed. For a working comparison of two
// builds use db-stats.js + verify-parity.js, which need no dependencies.
import Database from "better-sqlite3";

const NEW = new Database("public/thptqg2017.db", { readonly: true });
const OLD = new Database("backup/thptqg2017.old.db", { readonly: true });

const SCORE_COLS = [
  "toan", "ngu_van", "vat_ly", "hoa_hoc", "sinh_hoc", "khtn",
  "lich_su", "dia_ly", "gdcd", "khxh",
  "tieng_anh", "tieng_phap", "tieng_nga", "tieng_trung",
];

// Old schema may lack some newer columns — detect and project a subset
const oldCols = new Set(
  OLD.prepare("PRAGMA table_info(student)").all().map((c) => c.name),
);
const commonCols = SCORE_COLS.filter((c) => oldCols.has(c));

const newCount = NEW.prepare("SELECT COUNT(*) c FROM student").get().c;
const oldCount = OLD.prepare("SELECT COUNT(*) c FROM student").get().c;
console.log(`\n=== Row counts ===`);
console.log(`  new: ${newCount}`);
console.log(`  old: ${oldCount}`);
console.log(`  diff: ${newCount - oldCount >= 0 ? "+" : ""}${newCount - oldCount}`);

// Build SBD sets (memory: 861k strings ~50 MB, fine)
const newSbd = new Set(
  NEW.prepare("SELECT so_bao_danh FROM student").all().map((r) => r.so_bao_danh),
);
const oldSbd = new Set(
  OLD.prepare("SELECT so_bao_danh FROM student").all().map((r) => r.so_bao_danh),
);

let newOnly = 0, oldOnly = 0, common = 0;
for (const s of newSbd) (oldSbd.has(s) ? common++ : newOnly++);
for (const s of oldSbd) if (!newSbd.has(s)) oldOnly++;

console.log(`\n=== SBD membership ===`);
console.log(`  new-only (added):     ${newOnly}`);
console.log(`  old-only (removed):   ${oldOnly}`);
console.log(`  common:               ${common}`);

// Sample 5 new-only SBDs to peek
const sampleNewOnly = [];
for (const s of newSbd) {
  if (!oldSbd.has(s)) sampleNewOnly.push(s);
  if (sampleNewOnly.length >= 5) break;
}
if (sampleNewOnly.length) {
  const rows = NEW.prepare(
    `SELECT so_bao_danh, ho_ten FROM student WHERE so_bao_danh IN (${sampleNewOnly.map(() => "?").join(",")})`,
  ).all(...sampleNewOnly);
  console.log(`  sample new-only rows:`);
  rows.forEach((r) => console.log(`    ${r.so_bao_danh}  ${r.ho_ten}`));
}

// Score comparison on common SBDs
console.log(`\n=== Score comparison (common SBDs) ===`);
console.log(`  columns compared: ${commonCols.join(", ")}`);

const newStmt = NEW.prepare(
  `SELECT so_bao_danh, ${commonCols.join(",")} FROM student`,
);
const oldStmt = OLD.prepare(
  `SELECT so_bao_danh, ${commonCols.join(",")} FROM student`,
);

// Build old map
const oldMap = new Map();
for (const r of oldStmt.iterate()) oldMap.set(r.so_bao_danh, r);

let identical = 0,
  differ = 0;
const colChanges = Object.fromEntries(commonCols.map((c) => [c, { changed: 0, newNull: 0, oldNull: 0 }]));
const sampleDiffs = [];

for (const nr of newStmt.iterate()) {
  const or = oldMap.get(nr.so_bao_danh);
  if (!or) continue;
  let rowDiffers = false;
  const rowChanges = [];
  for (const c of commonCols) {
    const a = nr[c], b = or[c];
    if (a === b) continue;
    // treat very-close floats as equal (xlsx may emit 5.5 vs 5.50)
    if (a !== null && b !== null && Math.abs(a - b) < 1e-9) continue;
    rowDiffers = true;
    colChanges[c].changed++;
    if (a === null) colChanges[c].newNull++;
    if (b === null) colChanges[c].oldNull++;
    rowChanges.push(`${c}: ${b} → ${a}`);
  }
  if (rowDiffers) {
    differ++;
    if (sampleDiffs.length < 5) sampleDiffs.push({ sbd: nr.so_bao_danh, changes: rowChanges });
  } else {
    identical++;
  }
}

console.log(`  identical:  ${identical}`);
console.log(`  differing:  ${differ}`);
console.log(`  per-column changes:`);
for (const [c, info] of Object.entries(colChanges)) {
  if (info.changed === 0) continue;
  console.log(`    ${c.padEnd(14)} changed=${info.changed}  (new-null=${info.newNull}, old-null=${info.oldNull})`);
}
if (sampleDiffs.length) {
  console.log(`  sample diffs:`);
  sampleDiffs.forEach((d) => {
    console.log(`    SBD ${d.sbd}: ${d.changes.slice(0, 4).join("; ")}${d.changes.length > 4 ? " ..." : ""}`);
  });
}
