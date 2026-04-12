# Phase 03 — React Static Site

## Context Links
- [plan.md](./plan.md)
- [phase-01](./phase-01-project-scaffolding.md) — provides Vite+React scaffold
- [phase-02](./phase-02-excel-parser-db-builder.md) — provides .db file

## Overview
- **Priority**: P1
- **Status**: Pending
- **Description**: Vite+React+TypeScript site that loads SQLite .db client-side via sql.js for score lookup

## Key Insights

- .db file is ~50-60MB — must show loading progress to user
- sql.js requires WASM file — load from CDN (cdnjs) or bundle in public/
- All queries are readonly SELECT — no write operations
- Vietnamese UI — all labels, placeholders, messages in Vietnamese
- Two search modes: by name (ho_ten LIKE) and by exam ID (so_bao_danh exact match)

## Requirements

### Functional
- Load .db file on page load with progress indicator
- Search by số báo danh (exact match) or họ tên (partial match, case-insensitive)
- Display results in a clean table with all score columns
- Show "không tìm thấy" when no results
- Limit results to 50 rows (prevent rendering 800K rows)

### Non-Functional
- First meaningful paint < 2s (before DB loads)
- Search response < 100ms after DB is loaded
- Mobile-responsive layout
- Works offline after initial load (static site + cached .db)

## Architecture

```
Browser
  ├── index.html (Vite entry)
  ├── main.tsx → App
  │     ├── useSqlite hook (loads .db, exposes query fn)
  │     ├── SearchBar component (input + mode toggle)
  │     └── ResultTable component (score display)
  └── sql.js WASM (from CDN)
```

### Data Flow

```
Page Load
  → fetch('/thptqg2017.db') with progress tracking
  → initSqlJs({ locateFile: cdnjs url })
  → new SQL.Database(arrayBuffer)
  → DB ready, enable search

User Types Query
  → debounce 300ms
  → if mode=id:  SELECT * FROM student WHERE so_bao_danh = ?
  → if mode=name: SELECT * FROM student WHERE ho_ten LIKE ? LIMIT 50
  → render ResultTable with rows
```

## Related Code Files

**Create:**
- `web/src/app.tsx` — main app layout (~60 lines)
- `web/src/hooks/use-sqlite.ts` — sql.js loading + query hook (~70 lines)
- `web/src/components/search-bar.tsx` — search input + mode toggle (~40 lines)
- `web/src/components/result-table.tsx` — score results table (~60 lines)
- `web/src/types/student.ts` — TypeScript interface (~20 lines)
- `web/src/index.css` — minimal styling (~50 lines)

**Modify:**
- `web/src/main.tsx` — import App + CSS

## Implementation Steps

### 1. Define Student type

```typescript
// web/src/types/student.ts
export interface Student {
  so_bao_danh: string;
  ho_ten: string;
  ngay_sinh: string | null;
  toan: number | null;
  ngu_van: number | null;
  vat_ly: number | null;
  hoa_hoc: number | null;
  sinh_hoc: number | null;
  khtn: number | null;
  lich_su: number | null;
  dia_ly: number | null;
  gdcd: number | null;
  khxh: number | null;
  tieng_anh: number | null;
}
```

### 2. Implement use-sqlite hook

```typescript
// web/src/hooks/use-sqlite.ts
// States: loading (with progress %), ready, error
// On mount: fetch .db file → init sql.js → create Database instance
// Expose: { db, loading, progress, error, query(sql, params) }
```

Key details:
- Use `fetch()` with `response.body.getReader()` for progress tracking
- sql.js WASM from: `https://cdnjs.cloudflare.com/ajax/libs/sql.js/1.11.0/sql-wasm.wasm`
- Memoize the Database instance with useRef

### 3. Implement search-bar component

- Single text input with placeholder "Nhập số báo danh hoặc họ tên..."
- Auto-detect mode: if input is all digits → search by soBaoDanh; else → search by hoTen
- Debounce input by 300ms before triggering query
- Minimum 2 characters to trigger name search

### 4. Implement result-table component

- Responsive HTML table
- Columns: STT, Số báo danh, Họ tên, Ngày sinh, Toán, Ngữ văn, Vật lí, Hóa học, Sinh học, KHTN, Lịch sử, Địa lí, GDCD, KHXH, Tiếng Anh
- Show null scores as "-" (not 0)
- Highlight search match in name column (bold)
- "Không tìm thấy kết quả" message when empty

### 5. Implement app.tsx layout

```
<header>
  <h1>Tra cứu điểm thi THPT QG 2017</h1>
</header>
<main>
  {loading ? <ProgressBar /> : <SearchBar />}
  <ResultTable />
</main>
<footer>
  Dữ liệu sưu tầm, chỉ mang tính tham khảo
</footer>
```

### 6. Styling (index.css)

- Clean, minimal CSS — no framework needed for this scope
- CSS variables for colors
- Mobile-first responsive: table scrolls horizontally on small screens
- Loading bar animation

### 7. SQL query construction

```sql
-- By exam ID (exact)
SELECT * FROM student WHERE so_bao_danh = ?

-- By name (partial, accent-insensitive not feasible in SQLite, use LIKE)
SELECT * FROM student WHERE ho_ten LIKE ? LIMIT 50
-- param: `%${query}%`
```

Note: SQLite LIKE is case-insensitive for ASCII only. Vietnamese diacritics mean users must type exact accents. This is acceptable — Vietnamese users expect this.

## Todo List

- [ ] Create Student TypeScript interface
- [ ] Implement use-sqlite hook with progress tracking
- [ ] Implement search-bar with auto-detect mode + debounce
- [ ] Implement result-table with all score columns
- [ ] Implement app.tsx layout with loading state
- [ ] Add minimal CSS styling (mobile-responsive)
- [ ] Test with actual .db file from Phase 02
- [ ] Verify search by soBaoDanh returns exact match
- [ ] Verify search by hoTen returns partial matches (limit 50)

## Success Criteria

1. `npm run dev` shows app with loading indicator while .db fetches
2. Search by exact soBaoDanh returns single student with correct scores
3. Search by partial name returns up to 50 matching students
4. No results shows Vietnamese "không tìm thấy" message
5. Table is readable on mobile (horizontal scroll)
6. No console errors in Chrome/Firefox

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| .db fetch takes too long (50MB+) | High | Medium | Show progress bar; consider gzip in Phase 04 |
| sql.js WASM CDN unreachable | Low | High | Bundle WASM in public/ as fallback |
| Vietnamese text search limitations | Medium | Low | Document that exact diacritics required; acceptable UX |
| 800K row render if query too broad | Medium | Medium | LIMIT 50 on all queries; show "refine search" message |

## Security Considerations
- All data is public exam scores — no auth needed
- sql.js runs client-side only — no server attack surface
- No user data collected or stored

## Next Steps
- Phase 04 adds build optimization, gzip, and deploy configuration
