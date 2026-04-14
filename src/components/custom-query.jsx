import { useState, useCallback, useEffect } from "react";

const MAX_ROWS = 1000;

const PRESET_GROUPS = [
  {
    category: "Xếp hạng môn",
    queries: [
      {
        label: "Top 10 điểm Toán",
        sql: `SELECT so_bao_danh, ho_ten, ngay_sinh, toan
FROM student WHERE toan IS NOT NULL
ORDER BY toan DESC LIMIT 10`,
      },
      {
        label: "Top 10 KHTN",
        sql: `SELECT so_bao_danh, ho_ten, vat_ly, hoa_hoc, sinh_hoc, khtn
FROM student WHERE khtn IS NOT NULL
ORDER BY khtn DESC LIMIT 10`,
      },
      {
        label: "Top 10 KHXH",
        sql: `SELECT so_bao_danh, ho_ten, lich_su, dia_ly, gdcd, khxh
FROM student WHERE khxh IS NOT NULL
ORDER BY khxh DESC LIMIT 10`,
      },
      {
        label: "Thí sinh ≥ 9 điểm Toán",
        sql: `SELECT so_bao_danh, ho_ten, ngay_sinh, toan
FROM student WHERE toan >= 9
ORDER BY toan DESC LIMIT 50`,
      },
    ],
  },
  {
    category: "Long An (SBD 49xxx)",
    queries: [
      {
        label: "Top 10 Toán - Long An",
        sql: `SELECT so_bao_danh, ho_ten, ngay_sinh, toan
FROM student
WHERE so_bao_danh LIKE '49%' AND toan IS NOT NULL
ORDER BY toan DESC LIMIT 10`,
      },
      {
        label: "Top 10 khối A - Long An",
        sql: `SELECT so_bao_danh, ho_ten, toan, vat_ly, hoa_hoc,
  ROUND(toan + vat_ly + hoa_hoc, 2) AS tong_khoi_a
FROM student
WHERE so_bao_danh LIKE '49%'
  AND toan IS NOT NULL AND vat_ly IS NOT NULL AND hoa_hoc IS NOT NULL
ORDER BY tong_khoi_a DESC LIMIT 10`,
      },
      {
        // Per-student MAX across 49 official 2017 blocks (A00-A11, B00-B08,
        // C00-C20, D01-D15) restricted to subjects in our DB. SQLite's
        // MAX(x,y,...) scalar returns NULL if ANY arg is NULL (not just all),
        // so every block is wrapped in COALESCE(..., -1). NULLIF(..., -1)
        // then restores NULL for students whose data yields no valid block.
        label: "Top 100 điểm khối cao nhất - Long An",
        sql: `SELECT
  so_bao_danh,
  ho_ten,
  ngay_sinh,
  ROUND(NULLIF(MAX(
    COALESCE(toan+vat_ly+hoa_hoc, -1),        -- A00
    COALESCE(toan+vat_ly+tieng_anh, -1),      -- A01
    COALESCE(toan+vat_ly+sinh_hoc, -1),       -- A02
    COALESCE(toan+vat_ly+lich_su, -1),        -- A03
    COALESCE(toan+vat_ly+dia_ly, -1),         -- A04
    COALESCE(toan+hoa_hoc+lich_su, -1),       -- A05
    COALESCE(toan+hoa_hoc+dia_ly, -1),        -- A06
    COALESCE(toan+lich_su+dia_ly, -1),        -- A07
    COALESCE(toan+lich_su+gdcd, -1),          -- A08
    COALESCE(toan+dia_ly+gdcd, -1),           -- A09
    COALESCE(toan+vat_ly+gdcd, -1),           -- A10
    COALESCE(toan+hoa_hoc+gdcd, -1),          -- A11
    COALESCE(toan+hoa_hoc+sinh_hoc, -1),      -- B00
    COALESCE(toan+sinh_hoc+lich_su, -1),      -- B01
    COALESCE(toan+sinh_hoc+dia_ly, -1),       -- B02
    COALESCE(toan+sinh_hoc+ngu_van, -1),      -- B03
    COALESCE(toan+sinh_hoc+gdcd, -1),         -- B04
    COALESCE(toan+sinh_hoc+tieng_anh, -1),    -- B08
    COALESCE(ngu_van+lich_su+dia_ly, -1),     -- C00
    COALESCE(ngu_van+toan+vat_ly, -1),        -- C01
    COALESCE(ngu_van+toan+hoa_hoc, -1),       -- C02
    COALESCE(ngu_van+toan+lich_su, -1),       -- C03
    COALESCE(ngu_van+toan+dia_ly, -1),        -- C04
    COALESCE(ngu_van+vat_ly+hoa_hoc, -1),     -- C05
    COALESCE(ngu_van+vat_ly+sinh_hoc, -1),    -- C06
    COALESCE(ngu_van+vat_ly+lich_su, -1),     -- C07
    COALESCE(ngu_van+hoa_hoc+sinh_hoc, -1),   -- C08
    COALESCE(ngu_van+vat_ly+dia_ly, -1),      -- C09
    COALESCE(ngu_van+hoa_hoc+lich_su, -1),    -- C10
    COALESCE(ngu_van+sinh_hoc+lich_su, -1),   -- C12
    COALESCE(ngu_van+sinh_hoc+dia_ly, -1),    -- C13
    COALESCE(ngu_van+toan+gdcd, -1),          -- C14
    COALESCE(ngu_van+vat_ly+gdcd, -1),        -- C16
    COALESCE(ngu_van+hoa_hoc+gdcd, -1),       -- C17
    COALESCE(ngu_van+lich_su+gdcd, -1),       -- C19
    COALESCE(ngu_van+dia_ly+gdcd, -1),        -- C20
    COALESCE(toan+ngu_van+tieng_anh, -1),     -- D01
    COALESCE(toan+ngu_van+tieng_nga, -1),     -- D02
    COALESCE(toan+ngu_van+tieng_phap, -1),    -- D03
    COALESCE(toan+ngu_van+tieng_trung, -1),   -- D04
    COALESCE(toan+hoa_hoc+tieng_anh, -1),     -- D07
    COALESCE(toan+sinh_hoc+tieng_anh, -1),    -- D08
    COALESCE(toan+lich_su+tieng_anh, -1),     -- D09
    COALESCE(toan+dia_ly+tieng_anh, -1),      -- D10
    COALESCE(ngu_van+vat_ly+tieng_anh, -1),   -- D11
    COALESCE(ngu_van+hoa_hoc+tieng_anh, -1),  -- D12
    COALESCE(ngu_van+sinh_hoc+tieng_anh, -1), -- D13
    COALESCE(ngu_van+lich_su+tieng_anh, -1),  -- D14
    COALESCE(ngu_van+dia_ly+tieng_anh, -1)    -- D15
  ), -1), 2) AS diem_khoi_cao_nhat
FROM student
WHERE so_bao_danh LIKE '49%'
ORDER BY diem_khoi_cao_nhat IS NULL, diem_khoi_cao_nhat DESC
LIMIT 100`,
      },
    ],
  },
  {
    category: "Thống kê",
    queries: [
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
        label: "Số TS theo ngoại ngữ",
        sql: `SELECT
  SUM(CASE WHEN tieng_anh   IS NOT NULL THEN 1 ELSE 0 END) AS tieng_anh,
  SUM(CASE WHEN tieng_phap  IS NOT NULL THEN 1 ELSE 0 END) AS tieng_phap,
  SUM(CASE WHEN tieng_nga   IS NOT NULL THEN 1 ELSE 0 END) AS tieng_nga,
  SUM(CASE WHEN tieng_trung IS NOT NULL THEN 1 ELSE 0 END) AS tieng_trung
FROM student`,
      },
      {
        label: "KHTN vs KHXH",
        sql: `SELECT
  SUM(CASE WHEN khtn IS NOT NULL THEN 1 ELSE 0 END) AS chon_khtn,
  SUM(CASE WHEN khxh IS NOT NULL THEN 1 ELSE 0 END) AS chon_khxh,
  SUM(CASE WHEN khtn IS NOT NULL AND khxh IS NOT NULL THEN 1 ELSE 0 END) AS chon_ca_hai
FROM student`,
      },
    ],
  },
  {
    category: "Hệ thống",
    queries: [
      {
        label: "Schema bảng student",
        sql: `PRAGMA table_info(student)`,
      },
    ],
  },
];

const SCHEMA_PRESET = PRESET_GROUPS[PRESET_GROUPS.length - 1].queries[0];

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

  // Auto-run the schema preset the first time the tab is opened so the user
  // sees the student columns instead of a blank textarea.
  useEffect(() => {
    if (db && columns.length === 0 && !sql) {
      executeQuery(SCHEMA_PRESET.sql);
      setSql(SCHEMA_PRESET.sql);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [db]);

  return (
    <div className="custom-query">
      <div className="preset-groups">
        {PRESET_GROUPS.map((group) => (
          <div key={group.category} className="preset-group">
            <span className="preset-label">{group.category}</span>
            <div className="preset-list">
              {group.queries.map((p, i) => (
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
          </div>
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
