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
        // Top 100 Long An students by their best of 49 official 2017
        // admission blocks. Each (student, block) pair is materialised via
        // UNION ALL; ROW_NUMBER picks each student's #1 block; the outer
        // SELECT ranks across students. Result includes the winning block
        // code so the user sees which combination produced the score.
        label: "Top 100 điểm khối cao nhất - Long An",
        sql: `WITH per_block AS (
  SELECT so_bao_danh, ho_ten, ngay_sinh, 'A00' k, toan+vat_ly+hoa_hoc s FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A01', toan+vat_ly+tieng_anh   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A02', toan+vat_ly+sinh_hoc    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A03', toan+vat_ly+lich_su     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A04', toan+vat_ly+dia_ly      FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A05', toan+hoa_hoc+lich_su    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A06', toan+hoa_hoc+dia_ly     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A07', toan+lich_su+dia_ly     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A08', toan+lich_su+gdcd       FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A09', toan+dia_ly+gdcd        FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A10', toan+vat_ly+gdcd        FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'A11', toan+hoa_hoc+gdcd       FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B00', toan+hoa_hoc+sinh_hoc   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B01', toan+sinh_hoc+lich_su   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B02', toan+sinh_hoc+dia_ly    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B03', toan+sinh_hoc+ngu_van   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B04', toan+sinh_hoc+gdcd      FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'B08', toan+sinh_hoc+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C00', ngu_van+lich_su+dia_ly  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C01', ngu_van+toan+vat_ly     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C02', ngu_van+toan+hoa_hoc    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C03', ngu_van+toan+lich_su    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C04', ngu_van+toan+dia_ly     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C05', ngu_van+vat_ly+hoa_hoc  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C06', ngu_van+vat_ly+sinh_hoc FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C07', ngu_van+vat_ly+lich_su  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C08', ngu_van+hoa_hoc+sinh_hoc FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C09', ngu_van+vat_ly+dia_ly   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C10', ngu_van+hoa_hoc+lich_su FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C12', ngu_van+sinh_hoc+lich_su FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C13', ngu_van+sinh_hoc+dia_ly FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C14', ngu_van+toan+gdcd       FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C16', ngu_van+vat_ly+gdcd     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C17', ngu_van+hoa_hoc+gdcd    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C19', ngu_van+lich_su+gdcd    FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'C20', ngu_van+dia_ly+gdcd     FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D01', toan+ngu_van+tieng_anh  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D02', toan+ngu_van+tieng_nga  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D03', toan+ngu_van+tieng_phap FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D04', toan+ngu_van+tieng_trung FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D07', toan+hoa_hoc+tieng_anh  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D08', toan+sinh_hoc+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D09', toan+lich_su+tieng_anh  FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D10', toan+dia_ly+tieng_anh   FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D11', ngu_van+vat_ly+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D12', ngu_van+hoa_hoc+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D13', ngu_van+sinh_hoc+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D14', ngu_van+lich_su+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
  UNION ALL SELECT so_bao_danh, ho_ten, ngay_sinh, 'D15', ngu_van+dia_ly+tieng_anh FROM student WHERE so_bao_danh LIKE '49%'
),
ranked AS (
  SELECT *, ROW_NUMBER() OVER (PARTITION BY so_bao_danh ORDER BY s DESC, k) rn
  FROM per_block WHERE s IS NOT NULL
)
SELECT so_bao_danh, ho_ten, ngay_sinh, k AS khoi, ROUND(s, 2) AS diem
FROM ranked WHERE rn = 1
ORDER BY diem DESC LIMIT 100`,
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
