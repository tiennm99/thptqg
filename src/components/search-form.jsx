import { useEffect, useRef, useState } from "react";
import { detectMode } from "../lib/query-mode";

const DEBOUNCE_MS = 300;

export function SearchForm({
  value = "",
  onSearch,
  onClear,
  disabled,
  examples = [],
}) {
  const [query, setQuery] = useState(value);
  const inputRef = useRef(null);
  const timerRef = useRef(null);
  const { mode, hint } = detectMode(query);
  const canSearch = mode === "sbd" || mode === "name";

  // Sync from external value (deep-link URL, clear button in parent)
  useEffect(() => {
    setQuery(value);
  }, [value]);

  // Debounced live search
  useEffect(() => {
    clearTimeout(timerRef.current);
    if (!canSearch) {
      if (mode === "empty") onClear?.();
      return;
    }
    // Skip if the query already matches external value (prevents loop when
    // value flows in from URL hydration)
    if (query.trim() === value.trim() && query.trim() !== "") return;
    timerRef.current = setTimeout(() => onSearch(query.trim()), DEBOUNCE_MS);
    return () => clearTimeout(timerRef.current);
  }, [query, canSearch, mode, onSearch, onClear, value]);

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
            {examples.map((ex, i) => (
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
                {i < examples.length - 1 ? " hoặc " : ""}
              </span>
            ))}
          </>
        )}
      </p>
    </form>
  );
}
