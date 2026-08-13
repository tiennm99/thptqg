---
phase: 2
title: "Repo restructure"
status: completed
priority: P1
dependencies: [1]
effort: ""
---

# Phase 2: Repo restructure

## Overview

Move files into the target layout and migrate pnpm → npm. Relocation plus
package-manager swap — no application logic changes. Kept as its own commit so
the 419 MB data move is trivially revertable and reviewable separately from
behavior changes.

## Requirements

- All moves via `git mv` so blobs are reused and history follows
- Working tree builds after the move (paths updated, nothing dangling)
- Repository size does not grow
- npm is the only package manager; `package-lock.json` committed, no pnpm files remain

## Architecture

Git stores blobs content-addressed, so renaming 299 tracked data files
(~419 MB working-tree) adds no new objects. The cost is local I/O and one large
tree rewrite, not repository growth.

Data directory sizes being moved: `2016/data` 62 MB, `2017/data` 286 MB,
`2017/data-old` 39 MB, `2017/data-old2` 32 MB.

### pnpm → npm

A single package with 3 runtime and 9 dev dependencies, no workspace. pnpm's
advantages (strict resolution, shared store) buy nothing at this size, and two
pieces of scaffolding disappear with it:

- `pnpm-workspace.yaml` exists **only** to whitelist `better-sqlite3`'s native
  build (`allowBuilds`). npm runs postinstall by default, so the file has no npm
  equivalent — it is deleted, not translated. Verified: this is the file's
  entire content in both projects.
- The `pnpm/action-setup@v4` CI step and `cache: 'pnpm'` both drop out.

Lockfiles cannot be converted; `package-lock.json` is generated fresh from
`package.json`. Migration direction is safe: pnpm's strict `node_modules` layout
forbids phantom dependencies, so anything that resolved under pnpm also resolves
under npm's flat tree. The reverse would not hold.

Latent issue surfaced while checking: `2017/scripts/diff-datasets.js` imports
`better-sqlite3`, which is **not declared in any `package.json`** — that script
cannot run today without an ad-hoc install. Do not add the dependency; Phase 5
uses the built-in `node:sqlite` instead (verified working on Node 24, no flag).
Either port `diff-datasets.js` to `node:sqlite` in this phase or leave it broken
exactly as it is today and note it — do not silently half-fix it.

## Related Code Files

**Moves**

| From | To |
|---|---|
| `2016/tools/xlsxread/` | `parser/` |
| `2016/tools/xlsxread/configs/thptqg2016-data.toml` | `parser/configs/2016.toml` |
| `2017/tools/xlsxread/configs/thptqg2017-data.toml` | `parser/configs/2017.toml` |
| `2017/tools/xlsxread/configs/thptqg2017-data-old.toml` | `parser/configs/2017-old.toml` |
| `2017/tools/xlsxread/configs/thptqg2017-data-old2.toml` | `parser/configs/2017-old2.toml` |
| `2017/scripts/` | `parser/scripts/` |
| `2016/data/` | `data/2016/` |
| `2017/data/` | `data/2017/` |
| `2017/data-old/` | `data/2017-old/` |
| `2017/data-old2/` | `data/2017-old2/` |
| `2017/src/` | `src/` |
| `2017/index.html` | `index.html` (overwrites the old static landing page) |
| `2017/package.json`, `eslint.config.js`, `vite.config.js` | repo root |
| `2016/docs/*`, `2017/docs/*` | `docs/` |
| `2017/LICENSE` | `LICENSE` |

The old root `index.html` (the static hub) is **not moved** — it becomes
`src/components/hub.jsx` in Phase 3. Read it before deleting; its four links,
candidate counts, and Vietnamese copy are the source material for that
component.

**Deletes**
- `2016/` and `2017/` directories entirely (after moves)
- `2016/src/` — superseded by `src/` (its unique columns and SQL presets are ported in Phase 3)
- `2016/tools/xlsxread/configs/thptqg2017-*.toml` — test fixtures, superseded by real configs
- `2016/pnpm-lock.yaml`, `2017/pnpm-lock.yaml`, both `pnpm-workspace.yaml`
- Duplicate `2016/package.json`, `2016/eslint.config.js`, `2016/vite.config.js`, `2016/LICENSE`
- Root `index.html` (static hub) — only after its content is captured for `hub.jsx`

**Merges**
- `.gitignore` — one root file. 2016's is the verbose GitHub Node template,
  2017's is terse and accurate. Take 2017's as the base, add `parser/target/`,
  `.build/`, and `dist/`.
- `package.json` — root file keeps 2017's dependency set (identical to 2016's).
  Drop the `"packageManager": "pnpm@11.1.1"` field.

## Implementation Steps

1. Branch: `refactor/unify-frontend-and-schema`.
2. Copy the old root `index.html` content somewhere durable for Phase 3
   (`plans/reports/` or the phase-03 file itself) — it is the hub's source copy.
3. `git mv 2017/src src`, then the 2017 root config files, then
   `git mv 2017/index.html index.html`.
4. `git mv 2016/tools/xlsxread parser`, then rename the four configs.
5. `git mv 2017/scripts parser/scripts`.
6. Move the four data directories.
7. Merge docs into `docs/`, prefixing 2016-specific filenames where they collide
   (`deployment-guide.md` and `system-architecture.md` exist in both — read both
   before merging; they describe different pipelines).
8. Delete the emptied `2016/` and `2017/` trees plus both `pnpm-workspace.yaml`
   and both `pnpm-lock.yaml`.
9. Write the merged root `.gitignore`.
10. Drop `"packageManager"` from `package.json`; run `npm install` to generate
    `package-lock.json`; commit the lockfile.
11. Fix paths inside moved files: `parser/Cargo.toml` package name/paths,
    config `--input`/`--output` defaults, `parser/scripts/*.js` relative paths.
12. Replace `pnpm` with `npm run` in every `package.json` script body.
13. `cargo test` from `parser/` — must still be green.

## Tests / Validation

- `cargo test --manifest-path parser/Cargo.toml`
- `npm ci && npm run lint` at root — `npm ci` proves the lockfile is in sync
- `git status` shows renames (R), not delete+add pairs
- No file outside `plans/` still references `2016/tools`, `2017/tools`,
  `2017/src`, `2017/data`, or `pnpm` — grep to confirm

## Success Criteria

- [x] Target layout matches `plan.md` exactly
- [x] `2016/` and `2017/` no longer exist
- [x] Old hub page content captured before deletion
- [x] Git reports renames, repository size unchanged
- [x] `package-lock.json` committed; `npm ci` succeeds; zero pnpm files remain
- [x] `cargo test` green from the new location

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Old hub page deleted before its copy is captured | Step 2 runs before any move; content also recoverable from git history |
| Docs collide silently on merge | Read both copies before merging; two same-named files describe different pipelines |
| Git records delete+add instead of rename | Use `git mv`; verify with `git status` before commit |
| Stale path references in scripts/CI | Grep sweep in validation; Phase 4 rewrites the workflow |
| Fresh npm lockfile resolves different transitive versions than pnpm did | All deps are caret-ranged and already floating; `npm run lint` + the Phase 4 local build run are the check |
