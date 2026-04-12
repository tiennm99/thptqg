import { useState, useCallback } from "react";
import { useSqlite } from "./hooks/use-sqlite";
import { SearchForm } from "./components/search-form";
import { ScoreTable } from "./components/score-table";
import "./App.css";

const DB_URL = import.meta.env.BASE_URL + "thptqg2017.db.gz";
const MAX_RESULTS = 100;

function App() {
  const { db, loading, error, progress } = useSqlite(DB_URL);
  const [results, setResults] = useState(null);
  const [searchError, setSearchError] = useState(null);

  const handleSearch = useCallback(
    (query) => {
      if (!db) return;
      setSearchError(null);

      try {
        const isExamId = /^\d+$/.test(query);
        let stmt;

        if (isExamId) {
          // Exact match on số báo danh
          stmt = db.prepare(
            "SELECT * FROM student WHERE so_bao_danh = $q LIMIT $limit"
          );
          stmt.bind({ $q: query, $limit: MAX_RESULTS });
        } else {
          // LIKE search on name (case-insensitive with Vietnamese)
          stmt = db.prepare(
            "SELECT * FROM student WHERE ho_ten LIKE $q LIMIT $limit"
          );
          stmt.bind({ $q: `%${query}%`, $limit: MAX_RESULTS });
        }

        const rows = [];
        while (stmt.step()) {
          rows.push(stmt.getAsObject());
        }
        stmt.free();

        setResults(rows);
      } catch (err) {
        setSearchError(err.message);
      }
    },
    [db]
  );

  return (
    <div className="app">
      <header>
        <h1>Tra cứu điểm thi THPT Quốc gia 2017</h1>
        <p className="subtitle">Dữ liệu 847.349 thí sinh toàn quốc</p>
      </header>

      <main>
        {loading && (
          <div className="loading">
            <p>Đang tải cơ sở dữ liệu... {progress > 0 ? `${progress}%` : ""}</p>
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }} />
            </div>
          </div>
        )}

        {error && <p className="error">Lỗi: {error}</p>}

        <SearchForm onSearch={handleSearch} disabled={loading || !!error} />

        {searchError && <p className="error">Lỗi truy vấn: {searchError}</p>}

        <ScoreTable results={results} />

        {results && results.length >= MAX_RESULTS && (
          <p className="warning">
            Hiển thị tối đa {MAX_RESULTS} kết quả. Vui lòng tìm kiếm cụ thể hơn.
          </p>
        )}
      </main>

      <footer>
        <p>
          Nguồn: Sưu tầm từ trang báo thời đó · Dữ liệu chỉ mang tính tham khảo
        </p>
      </footer>
    </div>
  );
}

export default App;
