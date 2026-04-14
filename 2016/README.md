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
npm install
npm run build:db    # Parse data/*.xlsx → public/thptqg2016.db
npm run dev         # Vite dev server
npm run build       # Production bundle → dist/
npm run lint        # ESLint
```

The GitHub Actions workflow (`.github/workflows/deploy.yml`) builds the DB, gzips it, and deploys to GitHub Pages on every push to `main`.

## Tech stack

React 19 · Vite · sql.js (WASM) · better-sqlite3 (build-time only) · GitHub Pages

## Documentation

- [Project overview (PDR)](./docs/project-overview-pdr.md)
- [Codebase summary](./docs/codebase-summary.md)
- [System architecture](./docs/system-architecture.md)
- [Deployment guide](./docs/deployment-guide.md)

**Source**: Collected from news sites at the time · Data is for reference only.
