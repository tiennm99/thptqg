---
phase: 3
title: "Unified frontend and dataset registry"
status: completed
priority: P1
dependencies: [2]
effort: ""
---

# Phase 3: Unified frontend and dataset registry

## Overview

Make the single 2017-derived frontend serve every page on the site: the four
dataset views plus the `/thptqg/` hub. Three kinds of work — port 2016's only
unique feature (cluster + gender columns) into the shared components, extract
everything genuinely per-dataset into a runtime registry, and add a small
pathname router with a hub route.

## Context — what actually differs

The 2017 frontend is a superset in every respect except two columns. Verified by
reading both trees:

**2017 has, 2016 lacks:** debounced live search with input-mode hints
(`search-form.jsx`, 127 vs 46 lines), `?q=` URL deep links, `student-detail.jsx`
(183 lines, single-result view), `lib/admission-blocks.js` (49 blocks +
`scoreTier` ladder), all-NULL column hiding, `/` and Ctrl+Enter shortcuts,
footer total count, load progress bar, richer SQL presets (335 vs 219 lines).

**2016 has, 2017 lacks:** `ten_cum_thi` and `gioi_tinh` table columns with a
`cumthi-cell` title tooltip, and a simpler 3-tier `scoreClass` (superseded by
2017's 6-tier `scoreTier`).

## Requirements

**Functional**
- 2016 site keeps cluster + gender columns; 2017 sites must not show them
- 2016 site gains every 2017 feature listed above
- Per-dataset chrome (title, subtitle, source, DB size, search examples, SQL presets) is data, not code
- Admission-block computation works for both exam years without branching
- `/thptqg/` renders a hub listing the four datasets; no DB is fetched there
- Deep links (`?q=`) keep working on every dataset route

**Non-functional**
- Dataset identity resolves from `location.pathname` — no fetch, no env plumbing
- No routing library; five static routes do not justify a dependency
- Hub route must not pull the 47 MB DB into its critical path

## Architecture

### `src/datasets.js` — the registry

```js
export const DATASETS = [
  {
    id: "2016",              // === URL segment === data dir === config === db file
    dbSizeMb: 48,
    title: "Tra cứu điểm thi THPT Quốc gia 2016",
    subtitle: "Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh",
    source: "Bộ GD&ĐT",
    examples: ["<real 2016 SBD>", "Nguyễn Minh Tiến"],
    presets: PRESETS_2016,   // from 2016/src/components/custom-query.jsx
    blocks: BLOCKS_2016,     // see admission blocks below
  },
  { id: "2017",      /* presets: PRESETS_2017 */ },
  { id: "2017-old",  /* same presets as 2017, own label */ },
  { id: "2017-old2", /* same presets as 2017, own label */ },
];

export const pathOf = (d) => `${import.meta.env.BASE_URL}${d.id}/`;
export const dbOf   = (d) => `${import.meta.env.BASE_URL}db/${d.id}.db.gz`;
```

Plain runtime data — no `import.meta.env.VITE_DATASET` inlining, no build
variants. All four entries ship in the one bundle; the preset SQL totals a few
KB gzipped. Because URLs are flat, path and DB URL are *derived* from `id`
rather than stored, so a dataset cannot be misconfigured into pointing at the
wrong database.

### `src/router.js`

Flat URLs make this an exact match on a single segment — the segment **is** the
dataset ID:

```js
export function resolveRoute(pathname = location.pathname) {
  const seg = pathname
    .slice(import.meta.env.BASE_URL.length)   // strip "/thptqg/"
    .replace(/\/$/, "");
  return DATASETS.find((d) => d.id === seg) ?? null;   // null → hub
}
```

No prefix sorting, no ambiguity between `2017` and `2017-old` — that entire
class of bug is designed out by the flat scheme. No `react-router`.

### Legacy path redirects

The two old nested URLs are kept alive by a small map, so existing links and
bookmarks resolve instead of 404ing:

```js
const LEGACY = { "2017/old": "2017-old", "2017/old2": "2017-old2" };
```

On a legacy match, `history.replaceState` to the canonical flat URL **preserving
`location.search`** (the `?q=` deep link must survive), then resolve normally.
Phase 4 emits `index.html` at both legacy paths so Pages serves them at all.

Droppable if you'd rather let the old URLs 404 — it costs ~5 lines and two file
copies, and nothing else in the plan depends on it.

### `src/components/hub.jsx`

Renders the four dataset links from `DATASETS` via `pathOf()`. Content ported
from the old static root `index.html` (captured in Phase 2 step 2): heading, the
sql.js one-liner explanation, candidate counts, and the "phiên bản cũ" grouping
of the two 2017 variants. Styled with the app's existing CSS instead of the old
inline `<style>` block.

Note the links now point at `/thptqg/2017-old/`, not `/thptqg/2017/old/`.

`App.jsx` becomes: resolve route → hub, or dataset view. `useSqlite` is only
mounted on a dataset route, so the hub never touches the DB.

### Accepted trade-off

The hub was static HTML that painted instantly and worked without JS; it now
waits on the bundle (~150 KB gzipped). Accepted for the pipeline simplification.
If it ever matters, the four links can be inlined as static markup in
`index.html` so they paint pre-hydration.

### Column visibility — no new mechanism needed

`score-table.jsx` already computes `visibleColumns` by dropping columns where
every row is NULL. Extending that same filter to `ten_cum_thi` and `gioi_tinh`
makes them appear on 2016 and vanish on 2017 automatically, with no dataset
conditional in the component. This is the cheapest correct approach and it is
already the file's established pattern.

Identity columns need their own render path (they are text, not scored cells),
so add an `IDENTITY_COLUMNS` list alongside `SUBJECT_COLUMNS` and apply the same
all-NULL filter to both.

### Admission blocks — one union list

`computeBlocks()` already skips any block where a subject score is missing.
So a single list covering both years needs no branching: GDCD/KHTN blocks
self-exclude on 2016 rows, and the 2016-only foreign-language blocks
self-exclude on 2017 rows.

Add to `ADMISSION_BLOCKS`: `D05` (Toán+Văn+Đức), `D06` (Toán+Văn+Nhật), and the
other Đức/Nhật combinations from Circular 03/2017 that the current list drops
with the comment "neither language appears in any source file" — that comment
becomes false once 2016 data uses the same schema, so update it.

Verify the 2016 block list against the 2016 admission regulation rather than
assuming the 2017 circular's codes applied unchanged that year. If they differ
materially, key the block list by exam year via the registry's `blocks` field;
if they do not, drop that field and keep the single union list.

### Subject labels

`student-detail.jsx`'s `SUBJECT_LABELS`/`SUBJECT_ORDER` and `score-table.jsx`'s
`SUBJECT_COLUMNS` are two hand-maintained copies of the same subject list. Merge
into one `src/lib/subjects.js` exporting the 16-subject ordered list with labels;
both components consume it. This mirrors what Phase 1 does on the Rust side.

## Related Code Files

- Create: `src/datasets.js`
- Create: `src/router.js`
- Create: `src/components/hub.jsx`
- Create: `src/lib/subjects.js`
- Modify: `src/App.jsx` — resolve route; hub or dataset view; chrome from the registry entry
- Modify: `src/components/score-table.jsx` — add identity columns, use `subjects.js`
- Modify: `src/components/student-detail.jsx` — use `subjects.js`, show cluster/gender when present
- Modify: `src/components/search-form.jsx` — `EXAMPLES` from the active dataset
- Modify: `src/components/custom-query.jsx` — `PRESET_GROUPS` from the active dataset
- Modify: `src/lib/admission-blocks.js` — add Đức/Nhật blocks, update stale comment
- Modify: `src/App.css` — port `.cumthi-cell` from `2016/src/App.css`; add hub styles
- Reference (deleted in Phase 2): old root `index.html` for hub content,
  `2016/src/components/custom-query.jsx` for the 2016 preset SQL

## Implementation Steps

1. Extract `src/lib/subjects.js`; repoint both components at it.
2. Add `IDENTITY_COLUMNS` + all-NULL filtering to `score-table.jsx`; port the
   `.cumthi-cell` style.
3. Write `src/datasets.js` with all four entries. Lift the 2016 preset SQL
   verbatim from the old 2016 `custom-query.jsx` (cluster averages, gender
   breakdown, cluster+gender listing, language-count summary).
4. Note: the 2017 presets contain a hardcoded `so_bao_danh LIKE '49%'` Long An
   query. Keep it only in the 2017 entries; write a 2016 equivalent or drop it.
5. Write `src/router.js`: exact segment match, plus the legacy redirect map.
6. Write `src/components/hub.jsx` from the captured static hub content.
7. Restructure `App.jsx`: resolve route → hub or dataset view; mount `useSqlite`
   only on dataset routes; read chrome from the resolved entry.
8. Repoint `search-form.jsx` and `custom-query.jsx` at the active dataset.
9. Extend `ADMISSION_BLOCKS` with the Đức/Nhật blocks; verify against the 2016
   regulation before committing the list.
10. Show cluster/gender in `student-detail.jsx` when non-null.
11. `npm run lint`.

## Tests / Validation

Manual, against a local preview of the single build (no test harness exists in
this repo today):

- `/thptqg/` renders the hub; network tab shows **no** `.db.gz` request
- Each of the four dataset routes loads its own DB — confirm `2017-old` fetches
  `db/2017-old.db.gz`, not `db/2017.db.gz`
- `/thptqg/2017/old/?q=Nguyen` redirects to `/thptqg/2017-old/?q=Nguyen` with the
  query intact, and lands on the right dataset
- Hub links point at flat URLs
- 2016 route: cluster + gender columns render; KHTN/KHXH/GDCD/Nga columns absent
- 2017 routes: no cluster/gender columns; KHTN/KHXH/GDCD present
- Single-result search opens `student-detail` on all four
- `?q=` deep link hydrates search on all four
- SQL tab presets execute without error on their own dataset
- Admission blocks: a 2016 student shows D05/D06 where applicable and no GDCD
  blocks; a 2017 student shows GDCD blocks and no Đức/Nhật blocks
- `npm run lint` clean

## Success Criteria

- [x] One `src/` serving four datasets **and** the hub, zero `if (dataset === ...)` in components
- [x] Subject list defined once in `src/lib/subjects.js`
- [x] Hub route fetches no database
- [x] Dataset path and DB URL are derived from `id`, not stored per entry
- [x] Legacy `/2017/old/` and `/2017/old2/` redirect to flat URLs, `?q=` preserved
- [x] 2016 route renders cluster + gender; 2017 routes do not
- [x] 2016 route has live search, deep links, student detail, score tiers
- [x] Admission blocks correct for both exam years
- [x] No routing library added
- [x] `npm run lint` green

## Risk Assessment

| Risk | Mitigation |
|---|---|
| 2016 SQL presets lost when `2016/src` is deleted | Step 3 lifts them verbatim; recoverable from git history if ordering slips |
| 2017 block list assumed valid for 2016 | Step 9 requires checking the 2016 regulation; registry can key blocks per year if they differ |
| All-NULL filter hides a column on a legitimately sparse result set | Filter is per result set, matching today's 2017 behavior — accepted existing trade-off, not a new one |
| Hardcoded Long An preset leaks into 2016 | Explicit step 4 |
| Router sends `/2017-old/` to the `2017` dataset | Designed out: exact segment match on a flat scheme, no prefix logic exists |
| Existing links to nested URLs break | Legacy redirect map + Phase 4 stub pages; `?q=` preserved through the rewrite |
| Hub paints slower than the old static HTML | Accepted and documented; inline-links fallback available if it bites |
