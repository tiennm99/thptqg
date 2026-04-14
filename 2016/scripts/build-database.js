import XLSX from "xlsx";
import Database from "better-sqlite3";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ASSETS_DIR = path.join(__dirname, "..", "assets");
const DB_PATH = path.join(__dirname, "..", "public", "thptqg2016.db");

// Score patterns for the DIEM_THI string format
const SCORE_PATTERNS = {
  toan: /Toán:\s*([\d.]+)/,
  ngu_van: /Ngữ văn:\s*([\d.]+)/,
  vat_ly: /Vật lí:\s*([\d.]+)/,
  hoa_hoc: /Hóa học:\s*([\d.]+)/,
  sinh_hoc: /Sinh học:\s*([\d.]+)/,
  lich_su: /Lịch sử:\s*([\d.]+)/,
  dia_ly: /Địa lí:\s*([\d.]+)/,
  tieng_anh: /Tiếng Anh:\s*([\d.]+)/,
  tieng_phap: /Tiếng Pháp:\s*([\d.]+)/,
  tieng_duc: /Tiếng Đức:\s*([\d.]+)/,
  tieng_nhat: /Tiếng Nhật:\s*([\d.]+)/,
  tieng_trung: /Tiếng Trung:\s*([\d.]+)/,
};

const ALL_SCORE_FIELDS = Object.keys(SCORE_PATTERNS);

// Parse score text "Toán: 3.75  Ngữ văn: 5.00 ..." into { toan: 3.75, ... }
function parseScoreString(diemThi) {
  const scores = {};
  for (const [field, pattern] of Object.entries(SCORE_PATTERNS)) {
    const match = diemThi.match(pattern);
    if (match) scores[field] = parseFloat(match[1]);
  }
  return scores;
}

// Detect header row by checking for known column names
const KNOWN_HEADERS = new Set([
  "SOBAODANH", "SBD", "HO_TEN", "HOTEN", "HỌ TÊN",
  "NGAY_SINH", "TEN_CUMTHI", "GIOI_TINH", "DIEM_THI", "STT",
  "TOAN", "VAN", "LY", "HOA", "SINH ", "SU", "DIA",
]);

function isHeaderRow(row) {
  if (!row || row.length < 2) return false;
  const first = String(row[0] || "").trim().toUpperCase();
  return KNOWN_HEADERS.has(first);
}

// Detect which format a file uses based on its header row
function detectFormat(headerRow) {
  if (!headerRow) return null;
  const cols = headerRow.map((c) => String(c || "").trim().toUpperCase());

  // Format: SBD, HOTEN, TOAN, VAN, LY, HOA, SINH, SU, DIA, ...
  if (cols[0] === "SBD" && cols[2] === "TOAN") return "separate-scores";

  // Build a column index map for flexible column ordering
  const map = {};
  for (let i = 0; i < cols.length; i++) {
    const c = cols[i];
    if (c === "SOBAODANH" || c === "SBD") map.sbd = i;
    else if (c === "HO_TEN" || c === "HOTEN" || c === "HỌ TÊN") map.ho_ten = i;
    else if (c === "NGAY_SINH") map.ngay_sinh = i;
    else if (c === "TEN_CUMTHI") map.ten_cum_thi = i;
    else if (c === "GIOI_TINH") map.gioi_tinh = i;
    else if (c === "DIEM_THI") map.diem_thi = i;
  }

  if (map.sbd !== undefined && map.diem_thi !== undefined) {
    return { type: "mapped", map };
  }

  return null;
}

// Process a file with separate score columns (dhhanghai format)
function processSeparateScoresRow(row) {
  const sbd = String(row[0] || "").trim();
  const hoTen = String(row[1] || "").trim();
  if (!sbd || !hoTen) return null;

  return {
    so_bao_danh: sbd,
    ho_ten: hoTen,
    ngay_sinh: null,
    ten_cum_thi: null,
    gioi_tinh: null,
    toan: parseFloat(row[2]) || null,
    ngu_van: parseFloat(row[3]) || null,
    vat_ly: parseFloat(row[4]) || null,
    hoa_hoc: parseFloat(row[5]) || null,
    sinh_hoc: parseFloat(row[6]) || null,
    lich_su: parseFloat(row[7]) || null,
    dia_ly: parseFloat(row[8]) || null,
    // row[9]=NGOAINGUTN, row[10]=NGOAINGUTL, row[11]=NGOAINGU (total)
    tieng_anh: parseFloat(row[11]) || null,
    tieng_phap: null,
    tieng_duc: null,
    tieng_nhat: null,
    tieng_trung: null,
  };
}

// Process a row using the column map
function processMappedRow(row, map) {
  const sbd = String(row[map.sbd] || "").trim();
  const hoTen = String(row[map.ho_ten] || "").trim();
  if (!sbd || !hoTen) return null;

  // Skip leaked header rows
  const sbdUpper = sbd.toUpperCase();
  if (KNOWN_HEADERS.has(sbdUpper) || KNOWN_HEADERS.has(hoTen.toUpperCase())) return null;

  const ngaySinh = map.ngay_sinh !== undefined ? String(row[map.ngay_sinh] || "").trim() : null;
  const tenCumThi = map.ten_cum_thi !== undefined ? String(row[map.ten_cum_thi] || "").trim() : null;
  const rawGioiTinh = map.gioi_tinh !== undefined ? String(row[map.gioi_tinh] || "").trim() : null;
  // Normalize gender: only accept "Nam" or "Nữ"
  const gioiTinh = (rawGioiTinh === "Nam" || rawGioiTinh === "Nữ") ? rawGioiTinh : null;
  const diemThi = map.diem_thi !== undefined ? String(row[map.diem_thi] || "") : "";

  const scores = parseScoreString(diemThi);

  return {
    so_bao_danh: sbd,
    ho_ten: hoTen,
    ngay_sinh: ngaySinh || null,
    ten_cum_thi: tenCumThi || null,
    gioi_tinh: gioiTinh || null,
    ...Object.fromEntries(ALL_SCORE_FIELDS.map((f) => [f, scores[f] ?? null])),
  };
}

// Standard 6-column format without header: SBD, HO_TEN, NGAY_SINH, TEN_CUMTHI, GIOI_TINH, DIEM_THI
const DEFAULT_MAP = {
  sbd: 0, ho_ten: 1, ngay_sinh: 2, ten_cum_thi: 3, gioi_tinh: 4, diem_thi: 5,
};

function main() {
  fs.mkdirSync(path.dirname(DB_PATH), { recursive: true });
  if (fs.existsSync(DB_PATH)) fs.unlinkSync(DB_PATH);

  const db = new Database(DB_PATH);

  db.exec(`
    CREATE TABLE student (
      so_bao_danh TEXT PRIMARY KEY,
      ho_ten      TEXT NOT NULL,
      ngay_sinh   TEXT,
      ten_cum_thi TEXT,
      gioi_tinh   TEXT,
      toan        REAL,
      ngu_van     REAL,
      vat_ly      REAL,
      hoa_hoc     REAL,
      sinh_hoc    REAL,
      lich_su     REAL,
      dia_ly      REAL,
      tieng_anh   REAL,
      tieng_phap  REAL,
      tieng_duc   REAL,
      tieng_nhat  REAL,
      tieng_trung REAL
    );
    CREATE INDEX idx_ho_ten ON student(ho_ten);
    CREATE INDEX idx_ten_cum_thi ON student(ten_cum_thi);
  `);

  const insert = db.prepare(`
    INSERT OR REPLACE INTO student
      (so_bao_danh, ho_ten, ngay_sinh, ten_cum_thi, gioi_tinh,
       toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, lich_su, dia_ly,
       tieng_anh, tieng_phap, tieng_duc, tieng_nhat, tieng_trung)
    VALUES
      (@so_bao_danh, @ho_ten, @ngay_sinh, @ten_cum_thi, @gioi_tinh,
       @toan, @ngu_van, @vat_ly, @hoa_hoc, @sinh_hoc, @lich_su, @dia_ly,
       @tieng_anh, @tieng_phap, @tieng_duc, @tieng_nhat, @tieng_trung)
  `);

  // Collect all Excel files (.xlsx and .xls)
  const files = fs.readdirSync(ASSETS_DIR)
    .filter((f) => f.endsWith(".xlsx") || f.endsWith(".xls"))
    .map((f) => path.join(ASSETS_DIR, f));

  let totalRows = 0;
  let errorCount = 0;

  const insertAll = db.transaction((files) => {
    for (const file of files) {
      const basename = path.basename(file);
      let fileRows = 0;

      try {
        const wb = XLSX.readFile(file);
        const ws = wb.Sheets[wb.SheetNames[0]];
        const rows = XLSX.utils.sheet_to_json(ws, { header: 1 });
        if (rows.length === 0) continue;

        let startRow = 0;
        let format = null;

        if (isHeaderRow(rows[0])) {
          format = detectFormat(rows[0]);
          startRow = 1;
        }

        for (let i = startRow; i < rows.length; i++) {
          const row = rows[i];
          if (!row || row.length < 2) continue;

          try {
            let record;

            if (format === "separate-scores") {
              record = processSeparateScoresRow(row);
            } else if (format && format.type === "mapped") {
              record = processMappedRow(row, format.map);
            } else {
              // No header or unrecognized: assume standard 6-column order
              record = processMappedRow(row, DEFAULT_MAP);
            }

            if (!record) continue;
            insert.run(record);
            fileRows++;
          } catch {
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
