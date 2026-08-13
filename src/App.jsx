import { useState, useCallback, useEffect } from "react";
import { useSqlite } from "./hooks/use-sqlite";
import { SearchForm } from "./components/search-form";
import { ScoreTable } from "./components/score-table";
import { StudentDetail } from "./components/student-detail";
import { CustomQuery } from "./components/custom-query";
import "./App.css";

const DB_URL = import.meta.env.BASE_URL + "thptqg2017.db.gz";
const MAX_RESULTS = 100;
const DB_SIZE_MB = 47;

// Strip Vietnamese diacritics: "Nguyễn Bửu Lộc" → "nguyen buu loc"
function toAscii(str) {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/gi, "d")
    .toLowerCase();
}

function isAsciiOnly(str) {
  for (let i = 0; i < str.length; i++) {
    if (str.charCodeAt(i) > 127) return false;
  }
  return true;
}

// Sync query to URL (?q=...) without adding history entries
function writeUrlQuery(q) {
  const u = new URL(window.location.href);
  if (q) u.searchParams.set("q", q);
  else u.searchParams.delete("q");
  window.history.replaceState({}, "", u);
}

function readUrlQuery() {
  return new URL(window.location.href).searchParams.get("q") || "";
}

function App() {
  const { db, loading, error, progress } = useSqlite(DB_URL);
  const [results, setResults] = useState(null);
  const [searchError, setSearchError] = useState(null);
  const [activeTab, setActiveTab] = useState("search");
  // Query is owned here so we can bidirectionally bind it to the URL
  const [query, setQuery] = useState(() => readUrlQuery());
  const [totalCount, setTotalCount] = useState(null);

  const handleSearch = useCallback(
    (q) => {
      if (!db) return;
      setSearchError(null);
      setQuery(q);
      writeUrlQuery(q);

      try {
        const isExamId = /^\d+$/.test(q);
        let stmt;

        if (isExamId) {
          stmt = db.prepare(
            "SELECT * FROM student WHERE so_bao_danh = $q LIMIT $limit",
          );
          stmt.bind({ $q: q, $limit: MAX_RESULTS });
        } else if (isAsciiOnly(q)) {
          stmt = db.prepare(
            "SELECT * FROM student WHERE ho_ten_ascii LIKE $q LIMIT $limit",
          );
          stmt.bind({ $q: `%${toAscii(q)}%`, $limit: MAX_RESULTS });
        } else {
          const normalized = toAscii(q);
          stmt = db.prepare(
            `SELECT * FROM student
             WHERE ho_ten LIKE $q OR ho_ten_ascii LIKE $qn
             LIMIT $limit`,
          );
          stmt.bind({
            $q: `%${q}%`,
            $qn: `%${normalized}%`,
            $limit: MAX_RESULTS,
          });
        }

        const rows = [];
        while (stmt.step()) rows.push(stmt.getAsObject());
        stmt.free();
        setResults(rows);
      } catch (err) {
        setSearchError(err.message);
      }
    },
    [db],
  );

  // Run initial URL query once DB is ready
  useEffect(() => {
    if (db && query) handleSearch(query);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [db]);

  // Fetch total count for footer once DB loads
  useEffect(() => {
    if (!db) return;
    const stmt = db.prepare("SELECT COUNT(*) AS c FROM student");
    stmt.step();
    setTotalCount(stmt.getAsObject().c);
    stmt.free();
  }, [db]);

  // Global keyboard shortcuts:
  //   Ctrl+Enter → submit SQL query (when SQL tab active)
  //   "/"        → focus search box (when search tab active and not already typing)
  useEffect(() => {
    function handleKeyDown(e) {
      if (e.ctrlKey && e.key === "Enter" && activeTab === "sql") {
        const form = document.querySelector(".query-form");
        if (form) form.requestSubmit();
        return;
      }
      if (
        e.key === "/" &&
        activeTab === "search" &&
        !(e.target instanceof HTMLInputElement) &&
        !(e.target instanceof HTMLTextAreaElement)
      ) {
        e.preventDefault();
        document.getElementById("search-input")?.focus();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activeTab]);

  const handleClear = useCallback(() => {
    setResults(null);
    setQuery("");
    writeUrlQuery("");
  }, []);

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
            <p>
              Đang tải cơ sở dữ liệu ~{DB_SIZE_MB} MB
              {progress > 0 ? ` · ${progress}%` : ""}
            </p>
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }} />
            </div>
            <p className="loading-note">
              Lần đầu có thể mất 10-30 giây. Sau đó trình duyệt sẽ lưu cache
              và mở nhanh hơn.
            </p>
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
              value={query}
              onSearch={handleSearch}
              onClear={handleClear}
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
          Nguồn: baotintuc.vn
          {totalCount !== null && ` · ${totalCount.toLocaleString("vi-VN")} thí sinh`}
          {" · Dữ liệu chỉ mang tính tham khảo"}
        </p>
      </footer>
    </div>
  );
}

export default App;
