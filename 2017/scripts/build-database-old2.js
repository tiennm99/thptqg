// Build DB from data-old2/ — the 54 "update/" corrected-export files.
// Quirks:
//   • 54 provinces only (no Hà Nội, Bình Phước etc.)
//   • Mostly 2 sheets (one main + one empty trailing) — safe to iterate all
//   • HCM (24.HCM_UTLQ.xlsx) overflows into Sheet2 with +6,446 students —
//     multi-sheet walk is REQUIRED for it
//   • Some headers have extended columns beyond DIEM_THI (pre-parsed per-
//     subject cols). We still use only the DIEM_THI text at index 3.
import XLSX from "xlsx";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import {
  createDb,
  parseScores,
  isHeaderRow,
  buildRow,
} from "./build-lib.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC_DIR = path.join(__dirname, "..", "data-old2");
const DB_PATH = path.join(__dirname, "..", "public-old2", "thptqg2017.db");

function collectFiles() {
  return fs
    .readdirSync(SRC_DIR)
    .filter((f) => f.endsWith(".xlsx") || f.endsWith(".xls"))
    .map((f) => path.join(SRC_DIR, f));
}

function main() {
  const { db, insert } = createDb(DB_PATH);
  const files = collectFiles();
  let sourceRows = 0, skipped = 0, errors = 0;

  const run = db.transaction(() => {
    for (const file of files) {
      const base = path.basename(file);
      let fileRows = 0;
      const wb = XLSX.readFile(file);

      for (const sheetName of wb.SheetNames) {
        const rows = XLSX.utils.sheet_to_json(wb.Sheets[sheetName], {
          header: 1,
        });
        for (let i = 0; i < rows.length; i++) {
          if (i === 0 && isHeaderRow(rows[i])) continue;
          const r = rows[i];
          // Skip completely blank rows (common in xlsx tails) before counting
          if (!r || r.every((c) => c === "" || c == null)) continue;
          sourceRows++;
          const hoTen = String(r?.[0] || "").trim();
          const ngaySinh = String(r?.[1] || "").trim();
          const soBaoDanh = String(r?.[2] || "").trim();
          const diemThi = String(r?.[3] || "");
          if (!soBaoDanh || !hoTen) { skipped++; continue; }
          if (!/^\d+$/.test(soBaoDanh)) { skipped++; continue; }
          try {
            insert.run(buildRow({ hoTen, ngaySinh, soBaoDanh, scores: parseScores(diemThi) }));
            fileRows++;
          } catch (err) {
            errors++;
            if (errors <= 5) console.warn(`  [warn] ${base}: ${err.message}`);
          }
        }
      }
      console.log(`  ${base}: ${fileRows} rows`);
    }
  });

  console.log(`[build] data-old2/ → ${DB_PATH}  (${files.length} files)`);
  run();
  db.exec("VACUUM");

  const dbCount = db.prepare("SELECT COUNT(*) c FROM student").get().c;
  console.log(`\nSource non-blank data rows:      ${sourceRows}`);
  console.log(`  skipped (empty/non-numeric SBD): ${skipped}`);
  console.log(`  insertable:                      ${sourceRows - skipped}`);
  console.log(`  insert errors:                   ${errors}`);
  console.log(`DB rows (distinct SBD):            ${dbCount}`);
  const sz = fs.statSync(DB_PATH).size;
  console.log(`Size: ${(sz / 1024 / 1024).toFixed(1)} MB`);
  db.close();
}

main();
