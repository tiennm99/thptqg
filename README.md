# Vietnam THPT 2017 Score Lookup

Static React + SQLite site for looking up Vietnamese high school graduation exam scores (2017). The full database (~861k students, 63 provinces) ships to the browser as a compressed SQLite file and queries run client-side via `sql.js` — no backend.

## Live sites

Three deployments, one per dataset:

| URL | Data source | Rows |
|---|---|---|
| `/thptqg2017/`      | `data/` — [baotintuc.vn](https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm) CDN, `.xls` | **861,068** |
| `/thptqg2017/old/`  | `data-old/` — earlier `.xlsx` collection from Vietnamese news sites at the time (exact origin no longer known) | 847,348 |
| `/thptqg2017/old2/` | `data-old2/` — partial re-export (54 provinces) also from contemporary news sites, exact origin unrecorded | 679,764 |

## Features

- Diacritics-insensitive name search (`"nguyen"` matches `"Nguyễn"`)
- Live debounced search with URL deep-link (`?q=49008235`)
- Single-result detail card with per-subject TFT rarity-tiered scores (6 tiers, ≤1 → 9-10) and all 49 admission blocks (A00 – D15)
- Share button: copies a formatted summary plus deep-link URL
- SQL query tab with grouped presets (rankings, Long An filters, statistics, schema)
- Light + dark mode (follows OS preference)
- Keyboard shortcut `/` to focus search

## Requirements

- Node.js 20+
- npm

## Quickstart

```bash
npm install
npm run build:db     # parse data/ into public/thptqg2017.db (~2 min, 159 MB)
gzip -kf -9 public/thptqg2017.db
npm run dev          # http://localhost:5173
```

## Scripts

| Command | Action |
|---|---|
| `npm run dev` | Vite dev server |
| `npm run build` | Production build (main variant → `dist/`) |
| `npm run build:old` / `build:old2` | Build variant sites to `dist/old/`, `dist/old2/` |
| `npm run build:all` | All 3 web variants |
| `npm run build:db` | Build main DB from `data/` |
| `npm run build:db:old` / `build:db:old2` | Build old / old2 variant DBs |
| `npm run build:db:all` | All 3 DBs |
| `npm run lint` | ESLint |
| `node scripts/crawl-baotintuc.js` | Re-download all 63 province files from baotintuc.vn |
| `node scripts/audit-row-counts.js` | Verify source row count matches DB row count |
| `node scripts/check-duplicates.js` | MD5 + row-content duplicate audit |
| `node scripts/diff-datasets.js` | Compare `public/` vs `backup/` DB (when backup present) |

## Project layout

```
.
├── data/              # 63 .xls files (source)
├── data-old/          # 63 .xlsx (previous export)
├── data-old2/         # 54 .xlsx (update/ overrides)
├── public/            # main variant assets + thptqg2017.db.gz
├── public-old/        # old variant assets
├── public-old2/       # old2 variant assets
├── scripts/
│   ├── build-lib.js              # shared schema + helpers
│   ├── build-database.js         # parser for data/
│   ├── build-database-old.js     # parser for data-old/
│   ├── build-database-old2.js    # parser for data-old2/
│   ├── crawl-baotintuc.js        # downloader
│   ├── audit-row-counts.js       # parse-loss audit
│   ├── check-duplicates.js       # md5 dup detector
│   └── diff-datasets.js          # DB-to-DB comparator
├── src/
│   ├── App.jsx
│   ├── App.css
│   ├── components/{search-form, score-table, student-detail, custom-query}.jsx
│   ├── hooks/use-sqlite.js
│   └── lib/admission-blocks.js   # 49 admission-block definitions + score-tier helper
├── docs/              # see docs/README.md
├── index.html
├── vite.config.js
└── package.json
```

See `docs/` for architecture + deployment details.

## License

See `LICENSE`.

## Source

- **`data/`** — scraped in full from the baotintuc.vn 2017 announcement article:
  `https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm`
  Reproducible via `node scripts/crawl-baotintuc.js`.
- **`data-old/`** and **`data-old2/`** — collected from Vietnamese news sites at the
  time of the 2017 exam. The specific publisher URLs were not recorded and cannot
  be recovered now. These datasets are preserved for comparison and historical
  reference only; `data/` is the canonical source for the main deployment.

Intended for reference only.
