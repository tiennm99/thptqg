# Deployment Guide

## Automatic (recommended)

Push to `main` → GitHub Actions builds and deploys to GitHub Pages automatically.

Workflow: `.github/workflows/deploy.yml`

CI steps:
1. `npm ci`
2. `npm run build:db` — generate `public/thptqg2016.db` from `data/*.xlsx`
3. `gzip -k -9 public/thptqg2016.db` — max compression, keep original
4. `npm run build` — Vite bundles `dist/`
5. `rm -f dist/thptqg2016.db` — ship only the gzipped copy
6. `actions/upload-pages-artifact@v3` + `actions/deploy-pages@v4`

One-time setup: **Settings → Pages → Source: GitHub Actions**.

## Manual (local verification)

```bash
npm install
npm run build:db
gzip -k -9 public/thptqg2016.db     # Linux/macOS; on Windows use 7zip or WSL
npm run build
rm dist/thptqg2016.db               # optional, shrinks artifact
npm run preview                     # serve dist/ locally
```

Open <http://localhost:4173/thptqg2016/>.

## Base path

`vite.config.js` sets `base: "/thptqg2016/"`. If you fork under a different repo name, update this to match `<repo-name>` so assets resolve correctly on GitHub Pages.

## Updating data

1. Add the new Excel file to `data/`.
2. If its header is unfamiliar, open `scripts/build-database.js` and extend `detectFormat()` or the `processSeparateScoresRow` / `processMappedRow` helpers.
3. Run `npm run build:db` locally to check row counts and error skips.
4. Commit + push → CI redeploys.

## Troubleshooting

| Symptom | Typical cause |
|---------|---------------|
| Blank page, 404 on assets | `base` in `vite.config.js` doesn't match the repo name |
| `Failed to fetch database: 404` | gzip step skipped, or `.db.gz` removed from `dist/` |
| WASM fails to load | `sql.js.org` blocked / offline — self-host `sql-wasm.wasm` in `public/` and update `SQL_WASM_URL` in `use-sqlite.js` |
| Missing rows after build | Excel file has an unknown header — check console for `Failed to read` or `errorCount` |
