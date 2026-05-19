# thptqg2016

Lookup tool for Vietnam's 2016 National High School Graduation Exam (THPT Quốc gia) scores — 877,461 candidates nationwide.

Fully static app running entirely in the browser (SQLite via `sql.js`). No backend, no query logging.

## Features

- **Quick lookup** by exam ID (`số báo danh`) or full name (diacritics-insensitive)
- **Custom read-only SQL** (SELECT / PRAGMA / EXPLAIN / WITH) with 7 built-in preset queries
- **Complete data**: 12 subjects (Math, Literature, Physics, Chemistry, Biology, History, Geography, English, French, German, Japanese, Chinese), exam cluster, date of birth, gender
- Safety caps: 100 rows for lookup, 1000 rows for custom SQL
- Dark mode, `Ctrl+Enter` shortcut to run queries

## Demo

<https://tiennm99.github.io/thptqg2016/>

## Development

```bash
# Build the SQLite database from source Excel files (requires Rust stable)
pnpm run build:db

# Or build in two steps:
pnpm run build:rust          # compile the xlsxread binary
./tools/xlsxread/target/release/xlsxread build \
  --schema tools/xlsxread/configs/thptqg2016-data.toml \
  --input data \
  --output public/thptqg2016.db

pnpm run dev         # Vite dev server
pnpm run build       # Production bundle → dist/
pnpm run lint        # ESLint
```

The database is built by the `xlsxread` Rust binary (`tools/xlsxread/`), which
reads the 119 mixed `.xls`/`.xlsx` files and auto-detects the column layout per
file (`separate-scores`, `mapped`, or positional default). No Node.js Excel
library is required at build time.

The GitHub Actions workflow (`.github/workflows/deploy.yml`) compiles
`xlsxread`, builds the DB, gzips it, and deploys to GitHub Pages on every push
to `main`.

## Tech stack

React 19 · Vite · sql.js (WASM) · xlsxread (Rust, build-time) · GitHub Pages

## Documentation

- [Project overview (PDR)](./docs/project-overview-pdr.md)
- [Codebase summary](./docs/codebase-summary.md)
- [System architecture](./docs/system-architecture.md)
- [Deployment guide](./docs/deployment-guide.md)

**Source**: Originally published at <https://dtntbacgiang.edu.vn/tin-tuc/tin-tuc-su-kien/cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html> (currently inaccessible). Data is for reference only.
