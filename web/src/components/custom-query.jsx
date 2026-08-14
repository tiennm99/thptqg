import { useState, useCallback, useEffect } from "react";

const MAX_ROWS = 1000;

export function CustomQuery({ db, disabled, presets = [] }) {
  // By convention every preset list ends with a "Hệ thống" group whose first
  // query dumps the table schema. See lib/sql-presets.js.
  const schemaPreset = presets[presets.length - 1]?.queries[0];

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

      // Read-only statements only. The database is a per-browser copy, so this
      // guards the user's own session against a typo, not the server.
      const upper = trimmed.toUpperCase();
      const allowed = ["SELECT", "PRAGMA", "EXPLAIN", "WITH"];
      if (!allowed.some((kw) => upper.startsWith(kw))) {
        setQueryError(
          "Chỉ hỗ trợ truy vấn đọc (SELECT, PRAGMA, EXPLAIN, WITH).",
        );
        return;
      }

      // Cap an unbounded SELECT: the tables run to hundreds of thousands of
      // rows and rendering them all would lock the tab.
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
    if (db && schemaPreset && columns.length === 0 && !sql) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      executeQuery(schemaPreset.sql);
      setSql(schemaPreset.sql);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [db]);

  return (
    <div className="custom-query">
      <div className="preset-groups">
        {presets.map((group) => (
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
