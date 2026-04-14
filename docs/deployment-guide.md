# Deployment Guide

Deploys to GitHub Pages via `.github/workflows/deploy.yml`. Every push to `main` rebuilds and redeploys all 3 variants.

## What the workflow does

1. Checkout, setup Node 20, `npm ci`
2. `npm run build:db:all` — builds 3 SQLite DBs (main, old, old2)
3. Gzip each DB at level 9
4. `npm run build:all` — builds 3 Vite bundles into `dist/`, `dist/old/`, `dist/old2/`
5. Remove uncompressed `.db` files from `dist/` (only `.gz` ships)
6. `actions/upload-pages-artifact` + `actions/deploy-pages`

Total CI time ≈ 4–6 min (DB build dominates).

## Resulting URLs

- `https://<user>.github.io/thptqg2017/`
- `https://<user>.github.io/thptqg2017/old/`
- `https://<user>.github.io/thptqg2017/old2/`

## Local reproduction

```bash
npm ci
npm run build:db:all
gzip -kf -9 public/thptqg2017.db public-old/thptqg2017.db public-old2/thptqg2017.db
npm run build:all
npx serve dist    # or any static server
```

## Adding a new variant

1. Drop source Excel files into a new `data-vN/` folder
2. Add `public-vN/` to `.gitignore` patterns for the `.db` + `.db.gz`
3. Copy `scripts/build-database-old.js` → `scripts/build-database-vN.js`; update `SRC_DIR` / `DB_PATH`; adjust parse logic for the new quirks (multi-sheet? blank rows? numeric guard?)
4. Add variant to `VARIANT_CONFIG` in `vite.config.js`:
   ```js
   vN: { base: "/thptqg2017/vN/", publicDir: "public-vN", outDir: "dist/vN" }
   ```
5. Add `build:db:vN` and `build:vN` npm scripts; wire them into `build:db:all` and `build:all`
6. Update `.github/workflows/deploy.yml` — add gzip step + rm step for the new DB

## Notes

- **DB gzip is non-cacheable across deploys** — every rebuild produces a new `.db.gz` (not byte-identical due to SQLite page shuffling). First-visit users pay the 47 MB download; subsequent visits hit browser cache until the next deploy.
- **GitHub Pages file size limit**: individual files ≤ 100 MB. Main gzipped DB is ~47 MB — safe. Uncompressed 159 MB DB would not fit; that's why we ship `.gz` and the frontend decompresses via `DecompressionStream`.
- **No server-side compression assumption.** The app reads the `.gz` file directly (not `Content-Encoding: gzip`). GH Pages does not reliably gzip on-the-fly for arbitrary paths; shipping pre-gzipped bytes is deterministic.

## Rollback

Deploys are stateless snapshots. To rollback, revert the commit on `main` and push — the next workflow rebuilds the older state. There's no state to migrate.
