// Shared SQLite schema + score-text helpers for all 3 build pipelines.
// The parse LOOP for each dataset lives in its own build-database-*.js file
// (so source-specific quirks stay isolated), but the DB shape and regex
// dictionary are centralised here so the 3 DBs stay drop-in compatible with
// the same frontend.
import Database from "better-sqlite3";
import fs from "fs";

const NUM = "(\\d+(?:\\.\\d+)?)";
export const SCORE_PATTERNS = {
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

export function parseScores(diemThi) {
  const out = {};
  for (const [k, re] of Object.entries(SCORE_PATTERNS)) {
    const m = diemThi.match(re);
    if (m) out[k] = parseFloat(m[1]);
  }
  return out;
}

export function toAscii(str) {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/gi, "d")
    .toLowerCase();
}

export function isHeaderRow(row) {
  if (!row || row.length < 3) return false;
  const first = String(row[0] || "").toUpperCase();
  return first === "HO_TEN" || first === "HỌ TÊN" || first === "STT";
}

// Create empty DB file at dbPath with the canonical schema + prepared insert.
// Caller owns the returned handles and must close them.
export function createDb(dbPath) {
  fs.mkdirSync(dbPath.replace(/[^/\\]+$/, ""), { recursive: true });
  if (fs.existsSync(dbPath)) fs.unlinkSync(dbPath);
  const db = new Database(dbPath);
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
  return { db, insert };
}

// Build the @named params object the INSERT expects
export function buildRow({ hoTen, ngaySinh, soBaoDanh, scores }) {
  return {
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
  };
}
