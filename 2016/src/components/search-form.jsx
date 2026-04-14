import { useState } from "react";

export function SearchForm({ onSearch, disabled }) {
  const [query, setQuery] = useState("");

  function handleSubmit(e) {
    e.preventDefault();
    const trimmed = query.trim();
    if (trimmed) onSearch(trimmed);
  }

  return (
    <form onSubmit={handleSubmit} className="search-form" role="search">
      <div className="search-field">
        <label htmlFor="search-input" className="visually-hidden" hidden>
          Tìm theo số báo danh hoặc họ tên
        </label>
        <input
          id="search-input"
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Nhập SBD hoặc họ tên (VD: nguyen van a)..."
          disabled={disabled}
          autoFocus
          autoComplete="off"
          spellCheck={false}
        />
        {query && (
          <button
            type="button"
            className="search-clear"
            onClick={() => setQuery("")}
            aria-label="Xóa từ khóa"
            tabIndex={-1}
          >
            ×
          </button>
        )}
      </div>
      <button type="submit" disabled={disabled || !query.trim()}>
        Tra cứu
      </button>
    </form>
  );
}
