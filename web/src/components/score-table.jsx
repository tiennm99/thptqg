import { scoreTier } from "../lib/admission-blocks";
import { SUBJECTS, IDENTITY_COLUMNS, hasAnyValue } from "../lib/subjects";

function formatScore(val) {
  if (val === null || val === undefined) return "—";
  return Number(val).toFixed(2);
}

export function ScoreTable({ results }) {
  if (!results) return null;
  if (results.length === 0) {
    return <p className="no-results">Không tìm thấy kết quả.</p>;
  }

  // Drop columns where every row in the result set is NULL. This is what makes
  // one table serve both exam years: 2016 rows surface Cụm thi / GT / Đức /
  // Nhật and hide KHTN / KHXH / GDCD / Nga, and 2017 rows do the reverse — with
  // no dataset conditional anywhere in this component.
  const visibleIdentity = IDENTITY_COLUMNS.filter((col) =>
    hasAnyValue(results, col.key),
  );
  const visibleSubjects = SUBJECTS.filter((col) => hasAnyValue(results, col.key));

  return (
    <div className="table-wrapper">
      <p className="result-count">Tìm thấy {results.length} kết quả</p>
      <table>
        <thead>
          <tr>
            <th>SBD</th>
            <th>Họ tên</th>
            <th>Ngày sinh</th>
            {visibleIdentity.map((col) => (
              <th key={col.key}>{col.label}</th>
            ))}
            {visibleSubjects.map((col) => (
              <th key={col.key}>{col.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {results.map((row) => (
            <tr key={row.so_bao_danh}>
              <td className="sbd-cell">{row.so_bao_danh}</td>
              <td className="name-cell">{row.ho_ten}</td>
              <td>{row.ngay_sinh || "—"}</td>
              {visibleIdentity.map((col) => (
                <td
                  key={col.key}
                  className={col.key === "ten_cum_thi" ? "cumthi-cell" : undefined}
                  title={col.key === "ten_cum_thi" ? row[col.key] || "" : undefined}
                >
                  {row[col.key] || "—"}
                </td>
              ))}
              {visibleSubjects.map((col) => {
                const tier = scoreTier(row[col.key]);
                const cls = tier ? `score-cell tier-${tier.key}` : "score-cell";
                return (
                  <td
                    key={col.key}
                    className={cls}
                    title={tier ? tier.label : undefined}
                  >
                    {formatScore(row[col.key])}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
