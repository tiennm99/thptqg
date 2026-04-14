import { useState } from "react";

export function SearchForm({ onSearch, disabled }) {
  const [query, setQuery] = useState("");

  function handleSubmit(e) {
    e.preventDefault();
    const trimmed = query.trim();
    if (trimmed) onSearch(trimmed);
  }

  return (
    <form onSubmit={handleSubmit} className="search-form">
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Nhập số báo danh hoặc họ tên..."
        disabled={disabled}
        autoFocus
      />
      <button type="submit" disabled={disabled || !query.trim()}>
        Tra cứu
      </button>
    </form>
  );
}
