import { DATASETS, pathOf } from "../datasets";

/** The /thptqg/ landing route. No database is fetched here. */
export function Hub() {
  const base = import.meta.env.BASE_URL;

  return (
    <div className="hub">
      <header>
        <h1>Tra cứu điểm thi THPT Quốc gia</h1>
        <p className="subtitle">
          Tra cứu điểm thi tốt nghiệp THPT Quốc gia theo số báo danh — truy vấn
          SQL chạy hoàn toàn trên trình duyệt (sql.js).
        </p>
      </header>

      <main>
        <ul className="hub-list" role="list">
          {DATASETS.map((d) => (
            <li key={d.id}>
              <a className="hub-link" href={pathOf(d, base)}>
                <span className="hub-label">{d.label}</span>
                <span className="hub-blurb">{d.blurb}</span>
              </a>
            </li>
          ))}
        </ul>
      </main>

      <footer>
        <p>Dữ liệu chỉ mang tính tham khảo.</p>
      </footer>
    </div>
  );
}
