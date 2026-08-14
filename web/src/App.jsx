import { useState, useCallback, useEffect } from "react";
import { useSqlite } from "./hooks/use-sqlite";
import { SearchForm } from "./components/search-form";
import { ScoreTable } from "./components/score-table";
import { StudentDetail } from "./components/student-detail";
import { CustomQuery } from "./components/custom-query";
import { Hub } from "./components/hub";
import { dbOf } from "./datasets";
import { resolveRoute } from "./router";
import { isExamId, normaliseExamId } from "./lib/query-mode";
import "./App.css";

const MAX_RESULTS = 100;

// Strip Vietnamese diacritics: "Nguyễn Bửu Lộc" → "nguyen buu loc"
function toAscii(str) {
  return str
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
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

/** The score-lookup view for one dataset. */
function DatasetApp({ dataset }) {
  const { db, loading, error, progress } = useSqlite(
    dbOf(dataset, import.meta.env.BASE_URL),
  );
  const [results, setResults] = useState(null);
  const [searchError, setSearchError] = useState(null);
  const [activeTab, setActiveTab] = useState("search");
  // Owned here, not in SearchForm, so it can be bound to the URL both ways.
  const [query, setQuery] = useState(() => readUrlQuery());
  const [totalCount, setTotalCount] = useState(null);

  const handleSearch = useCallback(
    (q) => {
      if (!db) return;
      setSearchError(null);
      setQuery(q);
      writeUrlQuery(q);

      try {
        let stmt;

        if (isExamId(q)) {
          // Letter-prefixed 2016 IDs are stored upper-case; digits are unaffected.
          stmt = db.prepare(
            "SELECT * FROM student WHERE so_bao_danh = $q LIMIT $limit",
          );
          stmt.bind({ $q: normaliseExamId(q), $limit: MAX_RESULTS });
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

  // Hydrate a ?q= deep link as soon as the database is ready. Keyed on `db`
  // alone so it fires at most once per mount and cannot cascade.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (db && query) handleSearch(query);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [db]);

  // Candidate count for the footer. One-shot per mount: the count cannot change
  // without a new database.
  useEffect(() => {
    if (!db) return;
    const stmt = db.prepare("SELECT COUNT(*) AS c FROM student");
    stmt.step();
    // eslint-disable-next-line react-hooks/set-state-in-effect
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

  // The assembler copies one index.html to every route, so its static <title>
  // cannot name a dataset. Set the real one once the route is resolved.
  useEffect(() => {
    document.title = dataset.title;
  }, [dataset.title]);

  return (
    <div className="app">
      <header>
        <h1>{dataset.title}</h1>
        <p className="subtitle">{dataset.subtitle}</p>
        <p className="hub-back">
          <a href={import.meta.env.BASE_URL}>← Tất cả các kỳ thi</a>
        </p>
      </header>

      <main>
        {loading && (
          <div className="loading">
            <p>
              Đang tải cơ sở dữ liệu ~{dataset.dbSizeMb} MB
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
              examples={dataset.examples}
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
          <CustomQuery
            db={db}
            disabled={loading || !!error}
            presets={dataset.presets}
          />
        )}
      </main>

      <footer>
        <p>
          Nguồn:{" "}
          <a href={dataset.source} target="_blank" rel="noopener noreferrer">
            {dataset.source}
          </a>
          {totalCount !== null && ` · ${totalCount.toLocaleString("vi-VN")} thí sinh`}
          {" · Dữ liệu chỉ mang tính tham khảo"}
        </p>
      </footer>
    </div>
  );
}

function App() {
  // Resolved once at mount: every route is a full page load on GitHub Pages,
  // so there is no in-app navigation to react to.
  const [dataset] = useState(() => resolveRoute());

  return dataset ? <DatasetApp dataset={dataset} /> : <Hub />;
}

export default App;
