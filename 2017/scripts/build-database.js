import XLSX from "xlsx";
import Database from "better-sqlite3";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const RAW_DIR = path.join(__dirname, "..", "data", "raw");
const DB_PATH = path.join(__dirname, "..", "public", "thptqg2017.db");

// Score patterns matching the Java Converter.java regex
const SCORE_PATTERNS = {
  toan: /Toán:\s*(\d*\.\d*)/,
  ngu_van: /Ngữ văn:\s*(\d*\.\d*)/,
  vat_ly: /Vật lí:\s*(\d*\.\d*)/,
  hoa_hoc: /Hóa học:\s*(\d*\.\d*)/,
  sinh_hoc: /Sinh học:\s*(\d*\.\d*)/,
  khtn: /KHTN:\s*(\d*\.\d*)/,
  lich_su: /Lịch sử:\s*(\d*\.\d*)/,
  dia_ly: /Địa lí:\s*(\d*\.\d*)/,
  gdcd: /GDCD:\s*(\d*\.\d*)/,
  khxh: /KHXH:\s*(\d*\.\d*)/,
  tieng_anh: /Tiếng Anh:\s*(\d*\.\d*)/,
};

// Collect all .xlsx files: raw/ first, then raw/(update)/ to overwrite
function collectExcelFiles() {
  const files = [];

  // Main raw files first
  for (const f of fs.readdirSync(RAW_DIR)) {
    const full = path.join(RAW_DIR, f);
    if (fs.statSync(full).isFile() && f.endsWith(".xlsx")) {
      files.push(full);
    }
  }

  // Update files second (will overwrite via INSERT OR REPLACE)
  const updateDir = path.join(RAW_DIR, "update");
  if (fs.existsSync(updateDir)) {
    for (const f of fs.readdirSync(updateDir)) {
      const full = path.join(updateDir, f);
      if (fs.statSync(full).isFile() && f.endsWith(".xlsx")) {
        files.push(full);
      }
    }
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
      so_bao_danh TEXT PRIMARY KEY,
      ho_ten      TEXT NOT NULL,
      ngay_sinh   TEXT,
      toan        REAL,
      ngu_van     REAL,
      vat_ly      REAL,
      hoa_hoc     REAL,
      sinh_hoc    REAL,
      khtn        REAL,
      lich_su     REAL,
      dia_ly      REAL,
      gdcd        REAL,
      khxh        REAL,
      tieng_anh   REAL
    );
    CREATE INDEX idx_ho_ten ON student(ho_ten);
  `);

  const insert = db.prepare(`
    INSERT OR REPLACE INTO student
      (so_bao_danh, ho_ten, ngay_sinh, toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn, lich_su, dia_ly, gdcd, khxh, tieng_anh)
    VALUES
      (@so_bao_danh, @ho_ten, @ngay_sinh, @toan, @ngu_van, @vat_ly, @hoa_hoc, @sinh_hoc, @khtn, @lich_su, @dia_ly, @gdcd, @khxh, @tieng_anh)
  `);

  const files = collectExcelFiles();
  let totalRows = 0;
  let errorCount = 0;

  // Wrap all inserts in a single transaction for speed
  const insertAll = db.transaction((files) => {
    for (const file of files) {
      const basename = path.basename(file);
      let fileRows = 0;

      try {
        const wb = XLSX.readFile(file);
        const ws = wb.Sheets[wb.SheetNames[0]];
        const rows = XLSX.utils.sheet_to_json(ws, { header: 1 });

        for (let i = 0; i < rows.length; i++) {
          const row = rows[i];

          // Skip header rows
          if (i === 0 && isHeaderRow(row)) continue;

          try {
            const hoTen = String(row[0] || "").trim();
            const ngaySinh = String(row[1] || "").trim();
            const soBaoDanh = String(row[2] || "").trim();
            const diemThi = String(row[3] || "");

            // Validate: soBaoDanh should be numeric-like
            if (!soBaoDanh || !hoTen) continue;

            const scores = parseScores(diemThi);

            insert.run({
              so_bao_danh: soBaoDanh,
              ho_ten: hoTen,
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
            });
            fileRows++;
          } catch (err) {
            errorCount++;
          }
        }
      } catch (err) {
        console.error(`Failed to read ${basename}: ${err.message}`);
      }

      totalRows += fileRows;
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
