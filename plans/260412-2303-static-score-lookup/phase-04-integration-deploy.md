# Phase 04 — Integration + Deploy Config

## Context Links
- [plan.md](./plan.md)
- [phase-02](./phase-02-excel-parser-db-builder.md) — produces .db file
- [phase-03](./phase-03-react-static-site.md) — produces React app

## Overview
- **Priority**: P2
- **Status**: Pending
- **Description**: Wire everything together, optimize .db delivery, configure GitHub Pages deploy

## Key Insights

- .db file is ~50-60MB — gzip reduces SQLite files by ~70% typically → ~15-20MB
- GitHub Pages has 100MB file size limit; raw .db is borderline, gzipped is safe
- Vite build output goes to `web/dist/` — GitHub Actions deploys this folder
- No server needed — fully static

## Requirements

### Functional
- `npm run build:db` → `npm run build` produces deployable `web/dist/`
- .db file served with gzip (either pre-compressed or server-side)
- GitHub Actions workflow for automated deploy on push to main

### Non-Functional
- Total deploy artifact < 25MB (gzipped .db + app bundle)
- Build completes in CI in < 5 minutes

## Architecture

```
GitHub Push (main)
  → Actions workflow
    → npm ci
    → node scripts/build-database.js  (produces web/public/thptqg2017.db)
    → npm run build                    (Vite builds web/dist/)
    → deploy web/dist/ to GitHub Pages
```

### .db Delivery Optimization

**Option A — Pre-gzip (recommended):**
- Build script outputs `thptqg2017.db`
- Post-build step: `gzip -k web/public/thptqg2017.db` → produces `.db.gz`
- Vite copies `.db.gz` to `dist/`
- Client fetches `.db.gz`, decompresses with `DecompressionStream` or pako
- Pros: works on any static host, no server config needed

**Option B — Rely on server gzip:**
- Serve raw .db, let CDN/server gzip on-the-fly
- Pros: simpler client code
- Cons: GitHub Pages may not gzip .db extension; unreliable

**Decision: Option A** — pre-gzip with client-side decompress. Use browser-native `DecompressionStream` (supported in all modern browsers).

## Related Code Files

**Create:**
- `.github/workflows/deploy.yml` — GitHub Actions workflow (~40 lines)

**Modify:**
- `scripts/build-database.js` — add gzip step after DB creation
- `web/src/hooks/use-sqlite.ts` — fetch .db.gz, decompress, then init sql.js
- `package.json` — add `build:all` script combining db + vite build
- `.gitignore` — ensure web/dist/ and .db files are excluded

## Implementation Steps

### 1. Add gzip post-processing to build script

```javascript
// At end of scripts/build-database.js
const zlib = require('zlib');
const dbBuffer = fs.readFileSync('web/public/thptqg2017.db');
const gzipped = zlib.gzipSync(dbBuffer);
fs.writeFileSync('web/public/thptqg2017.db.gz', gzipped);
console.log(`Compressed: ${(gzipped.length / 1024 / 1024).toFixed(1)}MB`);
```

### 2. Update use-sqlite hook for gzip fetch

```typescript
// Fetch .db.gz instead of .db
const response = await fetch('/thptqg2017.db.gz');
const reader = response.body.getReader();
// ... read chunks with progress ...
const compressed = new Uint8Array(allChunks);

// Decompress using DecompressionStream
const ds = new DecompressionStream('gzip');
const writer = ds.writable.getWriter();
writer.write(compressed);
writer.close();
const decompressed = await new Response(ds.readable).arrayBuffer();

// Init sql.js with decompressed buffer
const db = new SQL.Database(new Uint8Array(decompressed));
```

### 3. Add combined build script

```json
{
  "scripts": {
    "build:db": "node scripts/build-database.js",
    "build:web": "vite build --config web/vite.config.ts",
    "build:all": "npm run build:db && npm run build:web",
    "dev": "vite --config web/vite.config.ts",
    "preview": "vite preview --config web/vite.config.ts"
  }
}
```

### 4. Create GitHub Actions deploy workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          lfs: true   # in case Excel files are in LFS
      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
      - run: npm ci
      - run: npm run build:all
      - uses: actions/upload-pages-artifact@v3
        with:
          path: web/dist
      - uses: actions/deploy-pages@v4
```

### 5. Configure Vite base path for GitHub Pages

```typescript
// web/vite.config.ts
export default defineConfig({
  base: '/thptqg2017/',  // matches GitHub repo name
  // ...
});
```

### 6. Update .gitignore

```
node_modules/
web/dist/
web/public/thptqg2017.db
web/public/thptqg2017.db.gz
```

### 7. End-to-end verification

1. Run `npm run build:all`
2. Run `npm run preview`
3. Open in browser, search for known student
4. Verify scores display correctly
5. Check network tab: .db.gz transfer size < 25MB

## Todo List

- [ ] Add gzip step to build-database.js
- [ ] Update use-sqlite hook for .db.gz fetch + decompress
- [ ] Add build:all npm script
- [ ] Configure Vite base path for GitHub Pages
- [ ] Create .github/workflows/deploy.yml
- [ ] Update .gitignore
- [ ] End-to-end test: build:all → preview → search
- [ ] Verify GitHub Actions workflow passes

## Success Criteria

1. `npm run build:all` produces `web/dist/` with all assets + .db.gz
2. .db.gz file < 25MB
3. `npm run preview` serves working site from dist/
4. GitHub Actions workflow deploys successfully
5. Live site at `https://{username}.github.io/thptqg2017/` is functional

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Excel files not in git (too large for checkout) | Medium | High | Check if LFS needed; or commit .db.gz directly and skip build:db in CI |
| DecompressionStream not supported in old browsers | Low | Low | 95%+ browser support; show "update browser" message for others |
| GitHub Pages deploy fails on large artifact | Low | Medium | Gzipped artifact should be well under limits |
| CI build timeout (Excel parsing slow) | Low | Low | 119 files in < 60s locally; CI has 6h limit |

## Backwards Compatibility

- Existing `database.sqlite` (97MB) preserved — not deleted or modified
- Java source code preserved — can be removed in separate cleanup PR
- New site deployed to GitHub Pages — no impact on existing setup

## Rollback Plan

- Revert the deploy.yml workflow → Pages stops updating
- Previous Pages deployment (if any) is preserved in GitHub Pages history
- Local: `git revert` the integration commit; Phases 1-3 artifacts still work standalone

## Next Steps (Future, out of scope)

- Remove Java source code and Gradle files (cleanup PR)
- Add statistics page (average scores per province)
- PWA support for offline access
