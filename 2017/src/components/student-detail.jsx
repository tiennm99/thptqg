import { useState } from "react";
import { computeBlocks, scoreTier } from "../lib/admission-blocks";

const SUBJECT_LABELS = {
  toan: "Toán",
  ngu_van: "Ngữ văn",
  vat_ly: "Vật lí",
  hoa_hoc: "Hóa học",
  sinh_hoc: "Sinh học",
  khtn: "KHTN",
  lich_su: "Lịch sử",
  dia_ly: "Địa lí",
  gdcd: "GDCD",
  khxh: "KHXH",
  tieng_anh: "Tiếng Anh",
  tieng_phap: "Tiếng Pháp",
  tieng_nga: "Tiếng Nga",
  tieng_trung: "Tiếng Trung",
};

const SUBJECT_ORDER = [
  "toan", "ngu_van",
  "vat_ly", "hoa_hoc", "sinh_hoc", "khtn",
  "lich_su", "dia_ly", "gdcd", "khxh",
  "tieng_anh", "tieng_phap", "tieng_nga", "tieng_trung",
];

function fmt(n) {
  return n === null || n === undefined ? "—" : Number(n).toFixed(2);
}

export function StudentDetail({ student }) {
  const [copied, setCopied] = useState(false);
  const blocks = computeBlocks(student);
  const subjects = SUBJECT_ORDER
    .filter((k) => student[k] !== null && student[k] !== undefined)
    .map((k) => ({ key: k, label: SUBJECT_LABELS[k], score: student[k] }));

  function copySbd() {
    navigator.clipboard.writeText(student.so_bao_danh).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  }

  return (
    <article className="detail-card" aria-label="Chi tiết điểm thí sinh">
      <header className="detail-header">
        <div className="detail-identity">
          <h2>{student.ho_ten}</h2>
          <dl className="detail-meta">
            <div>
              <dt>SBD</dt>
              <dd>
                <span className="mono">{student.so_bao_danh}</span>
                <button
                  type="button"
                  className="copy-btn"
                  onClick={copySbd}
                  aria-label="Sao chép số báo danh"
                >
                  {copied ? "✓ Đã chép" : "Chép"}
                </button>
              </dd>
            </div>
            {student.ngay_sinh && (
              <div>
                <dt>Ngày sinh</dt>
                <dd className="mono">{student.ngay_sinh}</dd>
              </div>
            )}
          </dl>
        </div>
      </header>

      <section aria-labelledby="subjects-heading">
        <h3 id="subjects-heading" className="section-title">Điểm môn thi</h3>
        <ul className="score-grid" role="list">
          {subjects.map((s) => {
            const tier = scoreTier(s.score);
            return (
              <li
                key={s.key}
                className={`score-tile tier-${tier.key}`}
                aria-label={`${s.label}: ${fmt(s.score)}, ${tier.label}`}
              >
                <span className="score-subject">{s.label}</span>
                <span className="score-value mono">{fmt(s.score)}</span>
                <span className="score-tier">
                  <span className="score-symbol" aria-hidden="true">{tier.symbol}</span>
                  {tier.label}
                </span>
              </li>
            );
          })}
        </ul>
      </section>

      {blocks.length > 0 && (
        <section aria-labelledby="blocks-heading">
          <h3 id="blocks-heading" className="section-title">
            Tổng điểm khối thi
            <span className="section-hint">
              (Chỉ tính những khối thí sinh có đủ 3 môn)
            </span>
          </h3>
          <ul className="block-list" role="list">
            {blocks.map((b) => (
              <li key={b.code} className="block-row">
                <span className="block-code">{b.code}</span>
                <span className="block-label">{b.label}</span>
                <span className="block-parts mono">
                  {b.parts.map((p) => fmt(p.score)).join(" + ")}
                </span>
                <span className="block-total mono">{b.total.toFixed(2)}</span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </article>
  );
}
