import XLSX from "xlsx";
import Database from "better-sqlite3";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const RAW_DIR = path.join(__dirname, "..", "data");
const DB_PATH = path.join(__dirname, "..", "public", "thptqg2017.db");

// Score patterns. Note: source data uses both "X.YZ" and "X" forms; allow optional decimal.
const NUM = "(\\d+(?:\\.\\d+)?)";
const SCORE_PATTERNS = {
  toan: new RegExp("Toán:\\s*" + NUM),
  ngu_van: new RegExp("Ngữ văn:\\s*" + NUM),
  vat_ly: new RegExp("Vật lí:\\s*" + NUM),
  hoa_hoc: new RegExp("Hóa học:\\s*" + NUM),
  sinh_hoc: new RegExp("Sinh học:\\s*" + NUM),
  khtn: new RegExp("KHTN:\\s*" + NUM),
  lich_su: new RegExp("Lịch sử:\\s*" + NUM),
  dia_ly: new RegExp("Địa lí:\\s*" + NUM),
  gdcd: new RegExp("GDCD:\\s*" + NUM),
  khxh: new RegExp("KHXH:\\s*" + NUM),
  tieng_anh: new RegExp("Tiếng Anh:\\s*" + NUM),
  tieng_phap: new RegExp("Tiếng Pháp:\\s*" + NUM),
  tieng_nga: new RegExp("Tiếng Nga:\\s*" + NUM),
  tieng_trung: new RegExp("Tiếng Trung:\\s*" + NUM),
};

// Strip Vietnamese diacritics for ASCII-insensitive search
function toAscii(str) {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/gi, "d")
    .toLowerCase();
}

// Collect all .xls and .xlsx files directly under data/
function collectExcelFiles() {
  const files = [];
  for (const f of fs.readdirSync(RAW_DIR)) {
    const full = path.join(RAW_DIR, f);
    if (!fs.statSync(full).isFile()) continue;
    if (f.endsWith(".xls") || f.endsWith(".xlsx")) files.push(full);
  }
  return files;
}

// Parse score text into an object of score fields
function parseScores(diemThi) {
  const scores = {};
  for (const [field, pattern] of Object.entries(SCORE_PATTERNS)) {
    const match = diemThi.match(pattern);
    if (match) {
      scores[field] = parseFloat(match[1]);
    }
  }
  return scores;
}

// Detect if first row is a header (not actual student data)
function isHeaderRow(row) {
  if (!row || row.length < 3) return false;
  const first = String(row[0] || "").toUpperCase();
  return first === "HO_TEN" || first === "HỌ TÊN" || first === "STT";
}

function main() {
  // Ensure output directory exists
  fs.mkdirSync(path.dirname(DB_PATH), { recursive: true });

  // Remove old database if exists
  if (fs.existsSync(DB_PATH)) fs.unlinkSync(DB_PATH);

  const db = new Database(DB_PATH);

  // Create table
  db.exec(`
    CREATE TABLE student (
      so_bao_danh   TEXT PRIMARY KEY,
      ho_ten        TEXT NOT NULL,
      ho_ten_ascii  TEXT NOT NULL,
      ngay_sinh     TEXT,
      toan          REAL,
      ngu_van       REAL,
      vat_ly        REAL,
      hoa_hoc       REAL,
      sinh_hoc      REAL,
      khtn          REAL,
      lich_su       REAL,
      dia_ly        REAL,
      gdcd          REAL,
      khxh          REAL,
      tieng_anh     REAL,
      tieng_phap    REAL,
      tieng_nga     REAL,
      tieng_trung   REAL
    );
    CREATE INDEX idx_ho_ten ON student(ho_ten);
    CREATE INDEX idx_ho_ten_ascii ON student(ho_ten_ascii);
  `);

  const insert = db.prepare(`
    INSERT OR REPLACE INTO student
      (so_bao_danh, ho_ten, ho_ten_ascii, ngay_sinh,
       toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn,
       lich_su, dia_ly, gdcd, khxh,
       tieng_anh, tieng_phap, tieng_nga, tieng_trung)
    VALUES
      (@so_bao_danh, @ho_ten, @ho_ten_ascii, @ngay_sinh,
       @toan, @ngu_van, @vat_ly, @hoa_hoc, @sinh_hoc, @khtn,
       @lich_su, @dia_ly, @gdcd, @khxh,
       @tieng_anh, @tieng_phap, @tieng_nga, @tieng_trung)
  `);

  const files = collectExcelFiles();
  let errorCount = 0;

  // Wrap all inserts in a single transaction for speed
  const insertAll = db.transaction((files) => {
    for (const file of files) {
      const basename = path.basename(file);
      let fileRows = 0;

      try {
        const wb = XLSX.readFile(file);
        // Iterate ALL sheets — .xls files over 65,536 rows (Hà Nội, HCM) split
        // into continuation sheets that we must not miss.
        const allRows = [];
        for (const sheetName of wb.SheetNames) {
          const rows = XLSX.utils.sheet_to_json(wb.Sheets[sheetName], {
            header: 1,
          });
          for (let i = 0; i < rows.length; i++) {
            // Skip header row on every sheet (continuation sheets may repeat it,
            // or start directly with data — isHeaderRow handles both)
            if (i === 0 && isHeaderRow(rows[i])) continue;
            allRows.push(rows[i]);
          }
        }

        for (let i = 0; i < allRows.length; i++) {
          const row = allRows[i];
          try {
            const hoTen = String(row?.[0] || "").trim();
            const ngaySinh = String(row?.[1] || "").trim();
            const soBaoDanh = String(row?.[2] || "").trim();
            const diemThi = String(row?.[3] || "");

            // Skip truly empty rows (common tail padding in .xls)
            if (!soBaoDanh || !hoTen) continue;

            const scores = parseScores(diemThi);

            insert.run({
              so_bao_danh: soBaoDanh,
              ho_ten: hoTen,
              ho_ten_ascii: toAscii(hoTen),
              ngay_sinh: ngaySinh || null,
              toan: scores.toan ?? null,
              ngu_van: scores.ngu_van ?? null,
              vat_ly: scores.vat_ly ?? null,
              hoa_hoc: scores.hoa_hoc ?? null,
              sinh_hoc: scores.sinh_hoc ?? null,
              khtn: scores.khtn ?? null,
              lich_su: scores.lich_su ?? null,
              dia_ly: scores.dia_ly ?? null,
              gdcd: scores.gdcd ?? null,
              khxh: scores.khxh ?? null,
              tieng_anh: scores.tieng_anh ?? null,
              tieng_phap: scores.tieng_phap ?? null,
              tieng_nga: scores.tieng_nga ?? null,
              tieng_trung: scores.tieng_trung ?? null,
            });
            fileRows++;
          } catch (err) {
            errorCount++;
            if (errorCount <= 5) {
              console.warn(`  [warn] ${basename} row ${i}: ${err.message}`);
            }
          }
        }
      } catch (err) {
        console.error(`Failed to read ${basename}: ${err.message}`);
      }

      console.log(`  ${basename}: ${fileRows} rows`);
    }
  });

  console.log(`Processing ${files.length} Excel files...\n`);
  insertAll(files);

  // Compact the database
  db.exec("VACUUM");

  const count = db.prepare("SELECT COUNT(*) as cnt FROM student").get();
  console.log(`\nDone! ${count.cnt} students in database.`);
  console.log(`Errors skipped: ${errorCount}`);
  console.log(`Output: ${DB_PATH}`);

  const stat = fs.statSync(DB_PATH);
  console.log(`Size: ${(stat.size / 1024 / 1024).toFixed(1)} MB`);

  db.close();
}

main();
