// Audit: compare expected unique SBDs from raw Excel files vs DB row count
import XLSX from "xlsx";
import Database from "better-sqlite3";
import fs from "fs";
import path from "path";

const RAW_DIR = "D:/tiennm99/thptqg2017/data";
const DB_PATH = "D:/tiennm99/thptqg2017/public/thptqg2017.db";

function collectFiles() {
  const out = [];
  for (const f of fs.readdirSync(RAW_DIR)) {
    const full = path.join(RAW_DIR, f);
    if (fs.statSync(full).isFile() && f.endsWith(".xlsx")) out.push(full);
  }
  return out;
}

function isHeader(row) {
  if (!row || row.length < 3) return false;
  const first = String(row[0] || "").toUpperCase();
  return first === "HO_TEN" || first === "HỌ TÊN" || first === "STT";
}

const allSbd = new Set();
let totalDataRows = 0,
  emptyName = 0,
  emptySbd = 0,
  bothEmpty = 0;

for (const file of collectFiles()) {
  const wb = XLSX.readFile(file);
  const ws = wb.Sheets[wb.SheetNames[0]];
  const rows = XLSX.utils.sheet_to_json(ws, { header: 1 });
  for (let i = 0; i < rows.length; i++) {
    const r = rows[i];
    if (i === 0 && isHeader(r)) continue;
    totalDataRows++;
    const hoTen = String(r?.[0] || "").trim();
    const sbd = String(r?.[2] || "").trim();
    if (!hoTen && !sbd) {
      bothEmpty++;
      continue;
    }
    if (!hoTen) emptyName++;
    if (!sbd) emptySbd++;
    if (sbd) allSbd.add(sbd);
  }
}

const db = new Database(DB_PATH, { readonly: true });
const dbCount = db.prepare("SELECT COUNT(*) c FROM student").get().c;

console.log("=== Source vs DB ===");
console.log(`Source: total data rows across all files: ${totalDataRows}`);
console.log(`Source: rows with empty name AND sbd (skipped): ${bothEmpty}`);
console.log(`Source: rows with missing name only: ${emptyName}`);
console.log(`Source: rows with missing sbd only: ${emptySbd}`);
console.log(`Source: distinct SBDs: ${allSbd.size}`);
console.log(`DB:     row count: ${dbCount}`);
console.log(
  `Match:  ${allSbd.size === dbCount ? "YES — all unique SBDs accounted for" : "NO — gap of " + (allSbd.size - dbCount)}`,
);
