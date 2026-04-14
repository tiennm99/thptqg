import { useEffect, useRef, useState } from "react";

const DEBOUNCE_MS = 300;
const MIN_SBD_DIGITS = 3;
const MIN_NAME_CHARS = 2;
const EXAMPLES = ["49008235", "Nguyễn Minh Tiến"];

// Detect input mode for dynamic hint shown under the field
function detectMode(raw) {
  const q = raw.trim();
  if (!q) return { mode: "empty", hint: "Gõ SBD (chữ số) hoặc họ tên để tìm" };
  if (/^\d+$/.test(q)) {
    return q.length >= MIN_SBD_DIGITS
      ? { mode: "sbd", hint: `Tìm theo số báo danh · khớp chính xác` }
      : { mode: "sbd-short", hint: `Cần ít nhất ${MIN_SBD_DIGITS} chữ số` };
  }
  return q.length >= MIN_NAME_CHARS
    ? { mode: "name", hint: "Tìm theo họ tên · không phân biệt dấu và hoa/thường" }
    : { mode: "name-short", hint: `Cần ít nhất ${MIN_NAME_CHARS} ký tự` };
}

export function SearchForm({ onSearch, onClear, disabled }) {
  const [query, setQuery] = useState("");
  const inputRef = useRef(null);
  const timerRef = useRef(null);
  const { mode, hint } = detectMode(query);
  const canSearch = mode === "sbd" || mode === "name";

  // Debounced live search
  useEffect(() => {
    clearTimeout(timerRef.current);
    if (!canSearch) {
      if (mode === "empty") onClear?.();
      return;
    }
    timerRef.current = setTimeout(() => onSearch(query.trim()), DEBOUNCE_MS);
    return () => clearTimeout(timerRef.current);
  }, [query, canSearch, mode, onSearch, onClear]);

  function handleSubmit(e) {
    e.preventDefault();
    if (canSearch) {
      clearTimeout(timerRef.current);
      onSearch(query.trim());
    }
  }

  function handleClear() {
    setQuery("");
    inputRef.current?.focus();
  }

  return (
    <form onSubmit={handleSubmit} className="search-form" role="search">
      <label htmlFor="search-input" className="search-label">
        Tra cứu điểm
      </label>
      <div className="search-input-row">
        <div className={`search-input-wrap mode-${mode}`}>
          <input
            id="search-input"
            ref={inputRef}
            type="search"
            inputMode="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Ví dụ: 47006585 hoặc Nguyen Van An"
            disabled={disabled}
            autoFocus
            autoComplete="off"
            spellCheck={false}
            aria-describedby="search-hint"
          />
          {query && (
            <button
              type="button"
              className="clear-btn"
              onClick={handleClear}
              aria-label="Xoá tìm kiếm"
              tabIndex={-1}
            >
              ×
            </button>
          )}
        </div>
        <button
          type="submit"
          className="primary-btn"
          disabled={disabled || !canSearch}
        >
          Tra cứu
        </button>
      </div>
      <p id="search-hint" className={`search-hint mode-${mode}`} aria-live="polite">
        {hint}
        {mode === "empty" && (
          <>
            {" · VD: "}
            {EXAMPLES.map((ex, i) => (
              <span key={ex}>
                <button
                  type="button"
                  className="example-btn"
                  onClick={() => {
                    setQuery(ex);
                    inputRef.current?.focus();
                  }}
                >
                  {ex}
                </button>
                {i < EXAMPLES.length - 1 ? " hoặc " : ""}
              </span>
            ))}
          </>
        )}
      </p>
    </form>
  );
}
