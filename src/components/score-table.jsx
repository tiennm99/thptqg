import { scoreTier } from "../lib/admission-blocks";

const SUBJECT_COLUMNS = [
  { key: "toan", label: "Toán" },
  { key: "ngu_van", label: "Ngữ văn" },
  { key: "vat_ly", label: "Vật lí" },
  { key: "hoa_hoc", label: "Hóa học" },
  { key: "sinh_hoc", label: "Sinh học" },
  { key: "khtn", label: "KHTN" },
  { key: "lich_su", label: "Lịch sử" },
  { key: "dia_ly", label: "Địa lí" },
  { key: "gdcd", label: "GDCD" },
  { key: "khxh", label: "KHXH" },
  { key: "tieng_anh", label: "Tiếng Anh" },
  { key: "tieng_phap", label: "Tiếng Pháp" },
  { key: "tieng_nga", label: "Tiếng Nga" },
  { key: "tieng_trung", label: "Tiếng Trung" },
];

function formatScore(val) {
  if (val === null || val === undefined) return "—";
  return Number(val).toFixed(2);
}

export function ScoreTable({ results }) {
  if (!results) return null;
  if (results.length === 0) {
    return <p className="no-results">Không tìm thấy kết quả.</p>;
  }

  // Hide columns where every row in the result set is null (e.g. unused foreign langs)
  const visibleColumns = SUBJECT_COLUMNS.filter((col) =>
    results.some((row) => row[col.key] !== null && row[col.key] !== undefined),
  );

  return (
    <div className="table-wrapper">
      <p className="result-count">Tìm thấy {results.length} kết quả</p>
      <table>
        <thead>
          <tr>
            <th>SBD</th>
            <th>Họ tên</th>
            <th>Ngày sinh</th>
            {visibleColumns.map((col) => (
              <th key={col.key}>{col.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {results.map((row) => (
            <tr key={row.so_bao_danh}>
              <td>{row.so_bao_danh}</td>
              <td className="name-cell">{row.ho_ten}</td>
              <td>{row.ngay_sinh || "—"}</td>
              {visibleColumns.map((col) => {
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
