import { useState, useCallback, useEffect } from "react";
import { useSqlite } from "./hooks/use-sqlite";
import { SearchForm } from "./components/search-form";
import { ScoreTable } from "./components/score-table";
import { StudentDetail } from "./components/student-detail";
import { CustomQuery } from "./components/custom-query";
import "./App.css";

const DB_URL = import.meta.env.BASE_URL + "thptqg2017.db.gz";
const MAX_RESULTS = 100;

// Strip Vietnamese diacritics: "Nguyễn Bửu Lộc" → "nguyen buu loc"
function toAscii(str) {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/gi, "d")
    .toLowerCase();
}

function isAsciiOnly(str) {
  // True if no Vietnamese diacritics or non-ASCII chars present
  for (let i = 0; i < str.length; i++) {
    if (str.charCodeAt(i) > 127) return false;
  }
  return true;
}

function App() {
  const { db, loading, error, progress } = useSqlite(DB_URL);
  const [results, setResults] = useState(null);
  const [searchError, setSearchError] = useState(null);
  const [activeTab, setActiveTab] = useState("search");

  const handleSearch = useCallback(
    (query) => {
      if (!db) return;
      setSearchError(null);

      try {
        const isExamId = /^\d+$/.test(query);
        let stmt;

        if (isExamId) {
          stmt = db.prepare(
            "SELECT * FROM student WHERE so_bao_danh = $q LIMIT $limit",
          );
          stmt.bind({ $q: query, $limit: MAX_RESULTS });
        } else if (isAsciiOnly(query)) {
          // ASCII input → match against folded column (handles "nguyen" = "Nguyễn")
          stmt = db.prepare(
            "SELECT * FROM student WHERE ho_ten_ascii LIKE $q LIMIT $limit",
          );
          stmt.bind({ $q: `%${toAscii(query)}%`, $limit: MAX_RESULTS });
        } else {
          // Vietnamese input → search both original and folded columns
          const normalized = toAscii(query);
          stmt = db.prepare(
            `SELECT * FROM student
             WHERE ho_ten LIKE $q OR ho_ten_ascii LIKE $qn
             LIMIT $limit`,
          );
          stmt.bind({
            $q: `%${query}%`,
            $qn: `%${normalized}%`,
            $limit: MAX_RESULTS,
          });
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
    [db],
  );

  // Ctrl+Enter submits the SQL query form when in SQL tab
  useEffect(() => {
    function handleKeyDown(e) {
      if (e.ctrlKey && e.key === "Enter" && activeTab === "sql") {
        const form = document.querySelector(".query-form");
        if (form) form.requestSubmit();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activeTab]);

  return (
    <div className="app">
      <header>
        <h1>Tra cứu điểm thi THPT Quốc gia 2017</h1>
        <p className="subtitle">
          Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh
        </p>
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

        <div className="tabs">
          <button
            className={`tab ${activeTab === "search" ? "active" : ""}`}
            onClick={() => setActiveTab("search")}
          >
            Tra cứu
          </button>
          <button
            className={`tab ${activeTab === "sql" ? "active" : ""}`}
            onClick={() => setActiveTab("sql")}
          >
            Truy vấn SQL
          </button>
        </div>

        {activeTab === "search" && (
          <>
            <SearchForm
              onSearch={handleSearch}
              onClear={() => setResults(null)}
              disabled={loading || !!error}
            />

            {searchError && (
              <p className="error">Lỗi truy vấn: {searchError}</p>
            )}

            {results && results.length === 1 ? (
              <StudentDetail student={results[0]} />
            ) : (
              <ScoreTable results={results} />
            )}

            {results && results.length >= MAX_RESULTS && (
              <p className="warning">
                Hiển thị tối đa {MAX_RESULTS} kết quả. Vui lòng tìm kiếm cụ thể hơn.
              </p>
            )}
          </>
        )}

        {activeTab === "sql" && (
          <CustomQuery db={db} disabled={loading || !!error} />
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
