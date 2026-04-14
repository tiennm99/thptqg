# Codebase Summary

## Directory layout

```
thptqg2016/
├── data/                       # Source Excel files (~100, mixed formats)
├── scripts/
│   └── build-database.js       # Parse Excel → SQLite (build-time, Node + better-sqlite3)
├── public/
│   └── thptqg2016.db           # Generated DB, gzipped during CI
├── src/
│   ├── main.jsx                # React entry
│   ├── App.jsx                 # Root: tabs, lookup logic, useSqlite wiring
│   ├── App.css / index.css     # Design tokens, dark mode, a11y styles
│   ├── hooks/
│   │   └── use-sqlite.js       # Fetch .db.gz + decompress + init sql.js
│   └── components/
│       ├── search-form.jsx     # Input for exam ID / full name
│       ├── score-table.jsx     # Result table for lookups
│       └── custom-query.jsx    # SQL editor + presets + result grid
├── .github/workflows/deploy.yml   # CI: build db → gzip → vite build → Pages
├── vite.config.js              # base: "/thptqg2016/"
└── eslint.config.js
```

## Key modules

### `scripts/build-database.js`
Build-time only. Reads every `.xlsx/.xls` in `data/`, detects the header format (three variants), parses the `DIEM_THI` string via regex or separate score columns, normalizes gender, derives a diacritics-stripped `ho_ten_ascii` column for accent-insensitive search, and inserts into SQLite with three indexes (`ho_ten`, `ho_ten_ascii`, `ten_cum_thi`).

### `src/hooks/use-sqlite.js`
Streams `.db.gz` with download progress, decompresses via `DecompressionStream("gzip")`, loads `sql.js` (WASM served from the `sql.js.org` CDN), and returns `{ db, loading, error, progress }`.

### `src/App.jsx`
Two tabs: **Lookup** and **Custom SQL**. Lookup auto-detects exam IDs (regex `^[A-Z]{2,4}\d+$`) vs names and picks one of three query paths: exact exam ID / ASCII LIKE / original + ASCII LIKE. Capped at 100 rows.

### `src/components/custom-query.jsx`
Whitelists leading keywords (`SELECT`, `PRAGMA`, `EXPLAIN`, `WITH`), auto-appends `LIMIT 1000` when missing, measures `performance.now()` execution time, and ships 7 preset analytics queries.

## `student` table schema

```sql
so_bao_danh  TEXT PRIMARY KEY         -- exam ID
ho_ten       TEXT NOT NULL            -- full name
ho_ten_ascii TEXT NOT NULL            -- diacritics stripped, lowercased
ngay_sinh    TEXT                     -- date of birth
ten_cum_thi  TEXT                     -- exam cluster name
gioi_tinh    TEXT                     -- "Nam" | "Nữ" | NULL
toan, ngu_van, vat_ly, hoa_hoc,       -- REAL (nullable) subject scores
sinh_hoc, lich_su, dia_ly,
tieng_anh, tieng_phap, tieng_duc,
tieng_nhat, tieng_trung
```

Indexes: `idx_ho_ten`, `idx_ho_ten_ascii`, `idx_ten_cum_thi`.

## Conventions

- JS/JSX filenames: **kebab-case** (e.g., `search-form.jsx`, `use-sqlite.js`)
- React components: `PascalCase` named exports
- UI strings: Vietnamese (target audience)
- Code comments: English; explain *why*, not *what*
