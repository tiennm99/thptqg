// Build DB from data-old/ — the original 63 xlsx files (pre-baotintuc refresh).
// Quirks: single-sheet workbooks; some files have no header row, so we
// isHeaderRow-test row 0 and skip only when it matches. A previous bug let a
// HO_TEN='SOBAODANH' header leak through — that's handled here by the
// hoTen/soBaoDanh validity check (trim + non-empty + soBaoDanh must be digits).
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
const SRC_DIR = path.join(__dirname, "..", "data-old");
const DB_PATH = path.join(__dirname, "..", "public-old", "thptqg2017.db");

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
      // data-old/: always read ONLY sheet 0 — these are xlsx and never hit the
      // 65k row cap, so any additional sheet is noise.
      const rows = XLSX.utils.sheet_to_json(wb.Sheets[wb.SheetNames[0]], {
        header: 1,
      });
      for (let i = 0; i < rows.length; i++) {
        if (i === 0 && isHeaderRow(rows[i])) continue;
        sourceRows++;
        const r = rows[i];
        const hoTen = String(r?.[0] || "").trim();
        const ngaySinh = String(r?.[1] || "").trim();
        const soBaoDanh = String(r?.[2] || "").trim();
        const diemThi = String(r?.[3] || "");
        // Extra guard specific to old data: reject rows where sbd is not
        // numeric or ho_ten literally says 'HO_TEN' (header leak from the
        // prior conversion pipeline).
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
      console.log(`  ${base}: ${fileRows} rows`);
    }
  });

  console.log(`[build] data-old/ → ${DB_PATH}  (${files.length} files)`);
  run();
  db.exec("VACUUM");

  const dbCount = db.prepare("SELECT COUNT(*) c FROM student").get().c;
  console.log(`\nSource data rows (post-header):  ${sourceRows}`);
  console.log(`  skipped (empty/non-numeric SBD): ${skipped}`);
  console.log(`  insertable:                      ${sourceRows - skipped}`);
  console.log(`  insert errors:                   ${errors}`);
  console.log(`DB rows (distinct SBD):            ${dbCount}`);
  const sz = fs.statSync(DB_PATH).size;
  console.log(`Size: ${(sz / 1024 / 1024).toFixed(1)} MB`);
  db.close();
}

main();
