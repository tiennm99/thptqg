import { useState, useCallback } from "react";

const MAX_ROWS = 1000;

const PRESET_QUERIES = [
  {
    label: "Top 10 điểm Toán cao nhất",
    sql: `SELECT so_bao_danh, ho_ten, ten_cum_thi, toan
FROM student WHERE toan IS NOT NULL
ORDER BY toan DESC LIMIT 10`,
  },
  {
    label: "Điểm trung bình theo cụm thi",
    sql: `SELECT ten_cum_thi,
  COUNT(*) AS so_luong,
  ROUND(AVG(toan), 2) AS tb_toan,
  ROUND(AVG(ngu_van), 2) AS tb_van,
  ROUND(AVG(tieng_anh), 2) AS tb_anh
FROM student
GROUP BY ten_cum_thi
ORDER BY so_luong DESC
LIMIT 20`,
  },
  {
    label: "Thống kê theo giới tính",
    sql: `SELECT gioi_tinh,
  COUNT(*) AS so_luong,
  ROUND(AVG(toan), 2) AS tb_toan,
  ROUND(AVG(ngu_van), 2) AS tb_van
FROM student
WHERE gioi_tinh IS NOT NULL
GROUP BY gioi_tinh`,
  },
  {
    label: "Thí sinh đạt 9+ điểm Toán",
    sql: `SELECT so_bao_danh, ho_ten, ten_cum_thi, gioi_tinh, toan
FROM student WHERE toan >= 9
ORDER BY toan DESC LIMIT 50`,
  },
  {
    label: "Phân bố điểm Toán",
    sql: `SELECT
  CASE
    WHEN toan < 1 THEN '0-1'
    WHEN toan < 2 THEN '1-2'
    WHEN toan < 3 THEN '2-3'
    WHEN toan < 4 THEN '3-4'
    WHEN toan < 5 THEN '4-5'
    WHEN toan < 6 THEN '5-6'
    WHEN toan < 7 THEN '6-7'
    WHEN toan < 8 THEN '7-8'
    WHEN toan < 9 THEN '8-9'
    ELSE '9-10'
  END AS khoang_diem,
  COUNT(*) AS so_luong
FROM student WHERE toan IS NOT NULL
GROUP BY khoang_diem
ORDER BY khoang_diem`,
  },
  {
    label: "Số thí sinh theo ngoại ngữ",
    sql: `SELECT
  SUM(CASE WHEN tieng_anh IS NOT NULL THEN 1 ELSE 0 END) AS tieng_anh,
  SUM(CASE WHEN tieng_phap IS NOT NULL THEN 1 ELSE 0 END) AS tieng_phap,
  SUM(CASE WHEN tieng_duc IS NOT NULL THEN 1 ELSE 0 END) AS tieng_duc,
  SUM(CASE WHEN tieng_nhat IS NOT NULL THEN 1 ELSE 0 END) AS tieng_nhat,
  SUM(CASE WHEN tieng_trung IS NOT NULL THEN 1 ELSE 0 END) AS tieng_trung
FROM student`,
  },
  {
    label: "Schema bảng student",
    sql: `PRAGMA table_info(student)`,
  },
];

export function CustomQuery({ db, disabled }) {
  const [sql, setSql] = useState("");
  const [columns, setColumns] = useState([]);
  const [rows, setRows] = useState([]);
  const [queryError, setQueryError] = useState(null);
  const [execTime, setExecTime] = useState(null);

  const executeQuery = useCallback(
    (queryStr) => {
      if (!db) return;
      setQueryError(null);
      setColumns([]);
      setRows([]);
      setExecTime(null);

      const trimmed = queryStr.trim();
      if (!trimmed) return;

      // Safety: only allow read-only statements
      const upper = trimmed.toUpperCase();
      const allowed = ["SELECT", "PRAGMA", "EXPLAIN", "WITH"];
      if (!allowed.some((kw) => upper.startsWith(kw))) {
        setQueryError(
          "Chỉ hỗ trợ truy vấn đọc (SELECT, PRAGMA, EXPLAIN, WITH).",
        );
        return;
      }

      // Auto-add LIMIT if user forgot
      let finalSql = trimmed;
      if (
        upper.startsWith("SELECT") &&
        !upper.includes("LIMIT") &&
        !upper.includes("PRAGMA")
      ) {
        finalSql = `${trimmed.replace(/;$/, "")} LIMIT ${MAX_ROWS}`;
      }

      try {
        const start = performance.now();
        const stmt = db.prepare(finalSql);
        const colNames = stmt.getColumnNames();
        const resultRows = [];

        let count = 0;
        while (stmt.step() && count < MAX_ROWS) {
          resultRows.push(stmt.get());
          count++;
        }
        stmt.free();

        const elapsed = performance.now() - start;
        setColumns(colNames);
        setRows(resultRows);
        setExecTime(elapsed.toFixed(1));
      } catch (err) {
        setQueryError(err.message);
      }
    },
    [db],
  );

  function handleSubmit(e) {
    e.preventDefault();
    executeQuery(sql);
  }

  function handlePreset(presetSql) {
    setSql(presetSql);
    executeQuery(presetSql);
  }

  return (
    <div className="custom-query">
      <div className="preset-list">
        <span className="preset-label">Mẫu truy vấn:</span>
        {PRESET_QUERIES.map((p, i) => (
          <button
            key={i}
            className="preset-btn"
            onClick={() => handlePreset(p.sql)}
            disabled={disabled}
          >
            {p.label}
          </button>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="query-form">
        <textarea
          value={sql}
          onChange={(e) => setSql(e.target.value)}
          placeholder={`Nhập truy vấn SQL...\nVí dụ: SELECT * FROM student WHERE toan >= 9 LIMIT 10`}
          disabled={disabled}
          rows={5}
          spellCheck={false}
        />
        <div className="query-actions">
          <button type="submit" disabled={disabled || !sql.trim()}>
            Thực thi (Ctrl+Enter)
          </button>
          {execTime !== null && (
            <span className="exec-time">
              {rows.length} kết quả · {execTime}ms
            </span>
          )}
        </div>
      </form>

      {queryError && <p className="error">Lỗi: {queryError}</p>}

      {columns.length > 0 && (
        <div className="table-wrapper">
          <table>
            <thead>
              <tr>
                {columns.map((col, i) => (
                  <th key={i}>{col}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, ri) => (
                <tr key={ri}>
                  {row.map((cell, ci) => (
                    <td key={ci} className="score-cell">
                      {cell === null ? "NULL" : String(cell)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length >= MAX_ROWS && (
            <p className="warning">
              Hiển thị tối đa {MAX_ROWS} kết quả. Thêm LIMIT để giới hạn.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
