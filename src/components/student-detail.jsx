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

// Visible tier legend — keeps symbol + range + label co-located so users
// don't have to hover tiles to decode colors.
const TIER_LEGEND = [
  { key: "common",    symbol: "·", range: "≤ 1",   label: "Điểm liệt" },
  { key: "uncommon",  symbol: "○", range: "< 5",   label: "Chưa đạt" },
  { key: "rare",      symbol: "◆", range: "5-6.5", label: "Trung bình" },
  { key: "epic",      symbol: "★", range: "6.5-8", label: "Khá" },
  { key: "legendary", symbol: "✦", range: "8-9",   label: "Giỏi" },
  { key: "prismatic", symbol: "❖", range: "9-10",  label: "Xuất sắc" },
];

export function StudentDetail({ student }) {
  const [copied, setCopied] = useState(null);  // null | 'sbd' | 'share' | 'url'
  const blocks = computeBlocks(student);
  const subjects = SUBJECT_ORDER
    .filter((k) => student[k] !== null && student[k] !== undefined)
    .map((k) => ({ key: k, label: SUBJECT_LABELS[k], score: student[k] }));

  function flash(kind) {
    setCopied(kind);
    setTimeout(() => setCopied(null), 1500);
  }

  function copySbd() {
    navigator.clipboard.writeText(student.so_bao_danh).then(() => flash("sbd"));
  }

  // Formatted summary suitable for pasting into Zalo / Messenger:
  //   Nguyễn Văn A (SBD 49008235)
  //   Toán 8.50 · Ngữ văn 7.25 · ...
  //   Khối A: 22.75
  //   Xem tại: https://...?q=49008235
  function shareSummary() {
    const lines = [];
    lines.push(`${student.ho_ten} (SBD ${student.so_bao_danh})`);
    lines.push(subjects.map((s) => `${s.label} ${fmt(s.score)}`).join(" · "));
    if (blocks.length > 0) {
      lines.push(
        blocks.slice(0, 3).map((b) => `Khối ${b.code} ${b.total.toFixed(2)}`).join(" · "),
      );
    }
    lines.push(
      `${window.location.origin}${window.location.pathname}?q=${student.so_bao_danh}`,
    );
    const text = lines.join("\n");

    if (navigator.share) {
      navigator.share({ text }).catch(() => {});
      return;
    }
    navigator.clipboard.writeText(text).then(() => flash("share"));
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
                  {copied === "sbd" ? "✓ Đã chép" : "Chép"}
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
        <div className="detail-actions">
          <button
            type="button"
            className="primary-btn"
            onClick={shareSummary}
            aria-label="Chia sẻ bảng điểm"
          >
            {copied === "share" ? "✓ Đã chép" : "Chia sẻ"}
          </button>
        </div>
      </header>

      <section aria-labelledby="subjects-heading">
        <h3 id="subjects-heading" className="section-title">
          Điểm môn thi
          <ul className="tier-legend" role="list" aria-label="Chú thích mức điểm">
            {TIER_LEGEND.map((t) => (
              <li key={t.key} className={`tier-legend-item tier-${t.key}`}>
                <span aria-hidden="true">{t.symbol}</span>
                <span className="tier-range">{t.range}</span>
                <span className="tier-name">{t.label}</span>
              </li>
            ))}
          </ul>
        </h3>
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
