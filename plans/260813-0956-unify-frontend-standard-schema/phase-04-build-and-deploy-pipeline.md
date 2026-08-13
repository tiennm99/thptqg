---
phase: 4
title: "Build and deploy pipeline"
status: pending
priority: P1
dependencies: [3]
effort: ""
---

# Phase 4: Build and deploy pipeline

## Overview

Rewire Vite, npm scripts, and GitHub Actions to produce the whole site from
**one** frontend build and **one** parser binary. Because the app now owns
routing (Phase 3), the four Vite build variants collapse into a single build
plus a copy step.

## Requirements

**Functional**
- One Vite build produces every page: hub + four dataset routes
- Flat published paths: `/thptqg/`, `/thptqg/2016/`, `/thptqg/2017/`, `/thptqg/2017-old/`, `/thptqg/2017-old2/`
- Legacy `/thptqg/2017/old/` and `/thptqg/2017/old2/` still resolve (redirect stubs)
- Deep links work on every route without a 404 fallback
- Uncompressed `.db` files never ship — only `.db.gz`

**Non-functional**
- One `cargo build`, one `npm ci`, one `vite build` per CI run
- npm only; no pnpm steps or caches remain

## Architecture

### Vite

One config. No variants, no `DATASET` env, no `emptyOutDir` ordering problem —
all three of those existed only to work around the missing router.

```js
export default defineConfig({
  plugins: [react()],
  base: "/thptqg/",
  publicDir: ".build/public",   // gitignored; holds db/*.db.gz only
});
```

### Why the entry-point copies work

With an absolute `base`, the emitted `index.html` references
`/thptqg/assets/index-HASH.js` no matter which directory it is served from. So
the same file is a valid entry point at every depth, and GitHub Pages serves
each as a directory index:

```bash
for ds in "${DATASETS[@]}"; do
  mkdir -p "_site/$ds" && cp dist/index.html "_site/$ds/index.html"
done
```

With flat URLs the loop iterates the same `DATASETS` array used to build the
databases — no path translation between dataset ID and URL path, because they
are the same string.

This is what removes the need for the usual `404.html` SPA-fallback hack — which
matters concretely here, because that hack rewrites the URL and would interfere
with the existing `?q=` deep-link handling.

Also emit `dist/index.html` as `_site/404.html` so unknown paths render the hub
instead of Pages' default 404.

### Database staging

The parser writes into `.build/public/db/`, gzips in place, and the raw `.db` is
deleted before Vite copies `publicDir`. Today's pipeline instead ships the
uncompressed DB into `dist` and deletes it afterwards (`rm -f dist/*.db`) —
staging makes shipping a 47 MB uncompressed file structurally impossible rather
than dependent on a cleanup step running.

### Workflow

Current workflow compiles the same Rust crate twice and runs two `pnpm install`s.
Collapse to one of each, then loop the four datasets:

```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '24'
    cache: 'npm'
    cache-dependency-path: package-lock.json

- name: Build databases
  run: |
    set -euo pipefail
    cargo build --release --manifest-path parser/Cargo.toml
    mkdir -p .build/public/db
    for ds in 2016 2017 2017-old 2017-old2; do
      ./parser/target/release/xlsxread build \
        --schema "parser/configs/$ds.toml" \
        --input  "data/$ds" \
        --output ".build/public/db/$ds.db"
      gzip -9 ".build/public/db/$ds.db"      # no -k: raw file must not survive
    done

- name: Build site
  run: |
    npm ci
    npm run build

- name: Assemble
  run: |
    set -euo pipefail
    mkdir -p _site && cp -r dist/* _site/
    cp dist/index.html _site/404.html
    # one entry point per dataset — ID and URL segment are the same string
    for ds in 2016 2017 2017-old 2017-old2; do
      mkdir -p "_site/$ds" && cp dist/index.html "_site/$ds/index.html"
    done
    # legacy nested URLs — router rewrites these to the flat form
    for legacy in 2017/old 2017/old2; do
      mkdir -p "_site/$legacy" && cp dist/index.html "_site/$legacy/index.html"
    done
```

The dataset list appears in both steps. Define it once as a job-level env var
(`DATASETS: "2016 2017 2017-old 2017-old2"`) rather than repeating the literal.

Cache changes: `Swatinem/rust-cache` workspaces → `parser`; `setup-node` cache →
`npm` keyed on `package-lock.json`; the `pnpm/action-setup` step is deleted.

## Related Code Files

- Modify: `vite.config.js` — single build, `base: /thptqg/`, `.build/public` publicDir
- Modify: `package.json` — the six `build:db*` and three `build:*` variant scripts
  collapse to `build:db`, `build`, `assemble`, `build:site`
- Create: `scripts/assemble-site.js` — copies the entry point to each route and
  refuses to ship an uncompressed database
- Modify: `.github/workflows/deploy-pages.yml` — single toolchain setup, npm
  caches, new assemble step
- Modify: `eslint.config.js` — Node globals for `scripts/**`, ignore `_site`
- Modify: `.gitignore` — add `.build/`, `_site/`, `parser/target/`, keep `dist/`

### Deviation: the dataset loop is a Node script, not workflow shell

The plan sketched a `for ds in 2016 2017 …` loop inline in the workflow, with a
note to hoist the list into a job-level env var. That would still have been a
second copy of the dataset list. `parser/scripts/build-db.js` and
`scripts/assemble-site.js` both import `DATASET_IDS` from `src/datasets.js`
instead, so the four IDs are declared exactly once for the frontend, the
database build and the site assembly. It also makes the whole pipeline runnable
locally with `npm run build:site`, which is how it was verified.

## Implementation Steps

1. Write the single-build `vite.config.js`.
2. Collapse `package.json` scripts. The current `build:old`/`build:old2` shell
   out through `node -e` + `spawnSync` purely to set an env var — both delete
   outright rather than getting converted.
3. Rewrite the workflow: one cargo build, one `npm ci`, one `npm run build`,
   dataset loop, new assemble step.
4. Update the Rust and npm caches; delete the pnpm setup step.
5. Run the whole pipeline locally — `cargo` and `npm` are both available — and
   inspect `_site/` before pushing.
6. Serve `_site/` locally and click through all five routes.

## Tests / Validation

- Local run produces `_site/index.html`, `_site/404.html`, and
  `index.html` under `2016/`, `2017/`, `2017-old/`, `2017-old2/`,
  plus legacy `2017/old/` and `2017/old2/`
- `find _site -name '*.db'` returns nothing (only `.db.gz` present)
- All emitted `index.html` files are byte-identical
- Asset URLs in them are absolute `/thptqg/assets/...`
- Serving `_site/` locally: `/thptqg/2017-old/?q=...` loads
  `db/2017-old.db.gz` and hydrates the query with no redirect
- `/thptqg/2017/old/?q=...` rewrites to the flat URL with the query intact
- Workflow run on the branch deploys all five flat URLs plus the two legacy paths

## Success Criteria

- [x] One `vite.config.js`, no build variants, no `DATASET` env
- [x] Workflow compiles Rust once, installs Node deps once, builds the site once
- [x] All five flat URLs served; deep links intact
- [x] Both legacy nested URLs resolve rather than 404
- [x] Dataset list written once (`src/datasets.js`), not repeated in the workflow
- [x] No SPA 404-redirect hack in the repo
- [x] No uncompressed DB anywhere in the artifact — enforced by the assemble step
- [x] Generated DBs live in gitignored `.build/`, not in source directories
- [x] Zero pnpm references in the workflow

## Verification limits

Route resolution is verified by serving the assembled artifact over HTTP and
checking every published URL returns 200 with correct absolute asset
references, plus that all entry points are byte-identical.

The routing **JavaScript** has not been executed. This workspace is headless
with no browser available, so hub-vs-dataset rendering and the legacy
`/2017/old/` → `/2017-old/` rewrite are verified by construction and by unit
tests of the pure functions, not by running in a browser. That check needs a
real browser.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| 47 MB uncompressed DB ships | `gzip -9` without `-k` leaves no raw file; validation greps `_site` for `*.db` |
| Relative asset path sneaks in and breaks nested entry points | Absolute `base`; validation asserts `/thptqg/assets/` in all five copies |
| Nested route 404s on Pages | Real `index.html` at each path, verified against a local static server before push |
| Total artifact size | Unchanged — all four DBs already ship today; no new Pages size exposure |
| Stale cache keys silently rebuild everything | Cosmetic; verify first workflow run's timing |
