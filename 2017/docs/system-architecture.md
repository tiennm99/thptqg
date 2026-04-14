# System Architecture

Static site. No backend. Browser downloads a compressed SQLite file at boot, then every query runs locally via `sql.js` (WASM).

## Data flow

```
Excel files (.xls / .xlsx)
        │
        ▼  build-database*.js (Node)
SQLite DB (public*/thptqg2017.db)
        │
        ▼  gzip -9
thptqg2017.db.gz (~47 MB for main variant)
        │
        ▼  Vite build copies public*/ into dist/
Static site on GitHub Pages
        │
        ▼  browser loads
sql.js (WASM) opens the .db → queries run client-side
```

## Three deployment variants

One repo → three independent sites, same frontend, different dataset.

| Variant | Route | Source dir | Public dir | Build cmd |
|---|---|---|---|---|
| main  | `/thptqg2017/`      | `data/`      | `public/`      | `npm run build` |
| old   | `/thptqg2017/old/`  | `data-old/`  | `public-old/`  | `npm run build:old` |
| old2  | `/thptqg2017/old2/` | `data-old2/` | `public-old2/` | `npm run build:old2` |

`vite.config.js` reads `process.env.VARIANT` and switches `base`, `publicDir`, `outDir` accordingly. `emptyOutDir` is on only for the main build so the variant builds merge into `dist/old/`, `dist/old2/` cleanly.

## Schema (all 3 DBs share this)

```sql
CREATE TABLE student (
  so_bao_danh   TEXT PRIMARY KEY,     -- exam ID (8 digits, first 2 = province)
  ho_ten        TEXT NOT NULL,
  ho_ten_ascii  TEXT NOT NULL,        -- NFD-stripped lowercase for accent-insensitive search
  ngay_sinh     TEXT,                 -- dd/mm/yyyy
  toan, ngu_van, vat_ly, hoa_hoc, sinh_hoc, khtn,
  lich_su, dia_ly, gdcd, khxh,
  tieng_anh, tieng_phap, tieng_nga, tieng_trung   REAL
);
CREATE INDEX idx_ho_ten        ON student(ho_ten);
CREATE INDEX idx_ho_ten_ascii  ON student(ho_ten_ascii);
```

No German / Japanese language columns — neither appears in any source file.

Each score column is `NULL` when the student didn't take that subject. A student's "admission block" total is only computed when all 3 required subjects are non-null.

## Parse quirks (why 3 builder scripts)

Each dataset had a different quirk. Rather than one mega-parser with mode flags, each builder script is ~70 lines and isolates its own workarounds.

| Dataset | Quirk | Mitigation |
|---|---|---|
| data/ (baotintuc .xls) | Hanoi + HCM overflow past the 65,536 row-per-sheet .xls limit | Iterate all `wb.SheetNames` |
| data-old/ (xlsx) | Single-sheet always; one bogus header row (`SOBAODANH`/`HO_TEN`) leaked into an earlier DB | Strict numeric-SBD guard rejects the leak |
| data-old2/ (xlsx) | HCM overflow + many blank trailing rows | Multi-sheet walk + blank-row pre-filter |

Shared concerns (regex, schema, `toAscii`, header detection) live in `scripts/build-lib.js`.

## Admission blocks

Vietnamese universities admit students based on 3-subject combinations called "admission blocks" (khối thi). `src/lib/admission-blocks.js` lists all 49 official 2017 blocks computable from our schema (A00–A11, B00–B08, C00–C20, D01–D15). Each entry is `{ code, subjects: [3 keys], label }`.

`computeBlocks(student)` returns only blocks where the student has all 3 subject scores, sorted by total desc. Used by the student detail card.

The SQL preset "Top 10 best-block in Long An" materialises all 49 blocks as a `UNION ALL` CTE, picks each student's best block via `ROW_NUMBER() OVER (PARTITION BY so_bao_danh ORDER BY s DESC, k)`, then ranks across students.

## Score tiers (UI coding)

6-level TFT rarity ladder (white → green → blue → purple → gold → prismatic). Defined in `scoreTier()` in `src/lib/admission-blocks.js`.

| Tier | Range | UI label (Vietnamese) | English |
|---|---|---|---|
| common    | ≤ 1   | Điểm liệt  | Disqualifying |
| uncommon  | < 5   | Chưa đạt   | Below passing |
| rare      | 5–6.5 | Trung bình | Average |
| epic      | 6.5–8 | Khá        | Good |
| legendary | 8–9   | Giỏi       | Very good |
| prismatic | 9–10  | Xuất sắc   | Excellent |

Color + icon + text label — never color-only.

## Frontend key behaviors

- Query state lives in `App.jsx` and is synced to URL `?q=...` via `history.replaceState` — bookmarkable, shareable.
- `SearchForm` is a controlled component that 300ms-debounces input changes and auto-triggers search once the query is long enough (3+ digits for SBD, 2+ chars for name).
- Detection: all-digits → exact `so_bao_danh` match; ASCII-only → folded `ho_ten_ascii LIKE`; has Vietnamese diacritics → search both `ho_ten` and folded column.
- Exactly 1 result → `StudentDetail` card; 0 or >1 → `ScoreTable`.
- Global `/` key focuses the search input when not already typing.
- Share button on the detail card uses Web Share API when present, else clipboard with a pre-formatted summary + deep-link URL.
