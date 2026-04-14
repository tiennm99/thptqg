const SUBJECT_COLUMNS = [
  { key: "toan", label: "Toán" },
  { key: "ngu_van", label: "Ngữ văn" },
  { key: "vat_ly", label: "Vật lí" },
  { key: "hoa_hoc", label: "Hóa học" },
  { key: "sinh_hoc", label: "Sinh học" },
  { key: "lich_su", label: "Lịch sử" },
  { key: "dia_ly", label: "Địa lí" },
  { key: "tieng_anh", label: "T.Anh" },
  { key: "tieng_phap", label: "T.Pháp" },
  { key: "tieng_duc", label: "T.Đức" },
  { key: "tieng_nhat", label: "T.Nhật" },
  { key: "tieng_trung", label: "T.Trung" },
];

function formatScore(val) {
  if (val === null || val === undefined) return "—";
  return Number(val).toFixed(2);
}

function scoreClass(val) {
  if (val === null || val === undefined) return "score-cell score-empty";
  const num = Number(val);
  if (num >= 8) return "score-cell score-high";
  if (num < 4) return "score-cell score-low";
  return "score-cell";
}

export function ScoreTable({ results }) {
  if (!results) return null;
  if (results.length === 0) {
    return <p className="no-results">Không tìm thấy kết quả.</p>;
  }

  return (
    <>
      <p className="result-count">Tìm thấy {results.length} kết quả</p>
      <div className="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>SBD</th>
              <th>Họ tên</th>
              <th>Ngày sinh</th>
              <th>Cụm thi</th>
              <th>GT</th>
              {SUBJECT_COLUMNS.map((col) => (
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
                <td className="cumthi-cell" title={row.ten_cum_thi || ""}>
                  {row.ten_cum_thi || "—"}
                </td>
                <td>{row.gioi_tinh || "—"}</td>
                {SUBJECT_COLUMNS.map((col) => (
                  <td key={col.key} className={scoreClass(row[col.key])}>
                    {formatScore(row[col.key])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
