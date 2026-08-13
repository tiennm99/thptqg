---
phase: 5
title: "Parity verification and docs"
status: completed
priority: P1
dependencies: [4]
effort: ""
---

# Phase 5: Parity verification and docs

## Overview

The release gate. Prove no data was lost or invented by the schema unification,
then update the documentation that describes a two-project repo which no longer
exists.

## Requirements

**Functional**
- Every dataset's row count and per-column non-NULL count matches the Phase 1 baseline
- Newly-added columns read exactly zero on datasets that never had them
- All four sites work end-to-end against their rebuilt DBs

**Non-functional**
- The comparison is a committed, re-runnable script, not a one-off shell session

## Architecture

### Why counts, not checksums

The schema changed shape, so the DB files cannot be byte-identical and a whole-file
hash is meaningless. The meaningful invariants are:

1. **Row count** per dataset — unchanged
2. **Per-column non-NULL count** for every column that existed before — unchanged
3. **Per-column non-NULL count** for every column newly added to a dataset — zero
4. **Value-level spot check** — a stable sample of SBDs compared field by field

Invariant 3 is what catches the union-regex risk flagged in Phase 1: if the
16-subject regex map starts matching text in 2016 files that the 12-subject map
ignored, `khtn`/`khxh`/`gdcd`/`tieng_nga` will be non-zero on 2016 and the gate
fails loudly rather than silently corrupting the dataset.

### Script

`parser/scripts/verify-parity.js` — reads
`plans/reports/parser-parity-baseline.json`, opens each rebuilt DB, recomputes
the same statistics, and exits non-zero on any mismatch with a per-column diff
table.

Use **`node:sqlite`** (`DatabaseSync`), not `better-sqlite3` or `sql.js`.
Verified working on this repo's Node 24 with no flag and no dependency:

```js
const { DatabaseSync } = require("node:sqlite");
```

That choice matters beyond convenience — `better-sqlite3` is a native module
whose postinstall is exactly what the deleted `pnpm-workspace.yaml` `allowBuilds`
entry existed to permit. Using the built-in keeps the dependency count at zero
and leaves nothing for a future package-manager change to trip over.

For invariant 4, sample deterministically — e.g. every SBD ending in `0000` —
so reruns compare the same students.

## Related Code Files

- Create: `parser/scripts/verify-parity.js`
- Reference: `plans/reports/parser-parity-baseline.json` (from Phase 1)
- Create: `plans/reports/parser-parity-result.md` (the gate's output)
- Modify: `README.md` — new layout, new commands, four datasets
- Modify: `docs/system-architecture.md` — merged, single-project architecture
- Modify: `docs/deployment-guide.md` — merged, new workflow
- Modify: `docs/data-pipeline.md` — canonical schema, one parser, four configs
- Delete: `docs/codebase-summary.md`, `docs/project-overview-pdr.md` if they
  describe only the old 2016 project — read before deciding

## Implementation Steps

1. Write `verify-parity.js`.
2. Run against all four rebuilt DBs; capture output to
   `plans/reports/parser-parity-result.md`.
3. Investigate any mismatch before touching docs. A failure here means Phase 1's
   schema or regex unification is wrong — fix it there, do not adjust the gate.
4. Manual pass on all four preview builds against the checklist in Phase 3.
5. Update `README.md`: layout tree, npm commands (`npm ci`, `npm run build`),
   parser invocation, the four dataset descriptions and their flat URLs. Call
   out the URL change from `/2017/old/` to `/2017-old/` and that the old paths
   redirect.
6. Merge the duplicated docs. `deployment-guide.md` and `system-architecture.md`
   exist in both old projects and describe different pipelines — read both
   copies fully before writing the merged version.
7. Rewrite `docs/data-pipeline.md` around the canonical schema: the 22 columns,
   which datasets populate which, and how `format_detection` selects the 2016
   column-layout path.
8. Document the canonical schema itself in one place, referenced from the others.

## Tests / Validation

- `node parser/scripts/verify-parity.js` exits 0
- Row counts: 2016 ≈ 877,461 (per the current landing page), 2017 ≈ 861,000 —
  confirm against the baseline, not against these approximations
- 2016 DB: `khtn`, `khxh`, `gdcd`, `tieng_nga` all read 0 non-NULL
- 2017 DBs: `ten_cum_thi`, `gioi_tinh`, `tieng_duc`, `tieng_nhat` all read 0 non-NULL
- All four dataset routes: search by SBD, search by name, deep link, SQL preset,
  student detail; plus the hub route linking to all four
- No doc references `2016/tools`, `2017/src`, `public-old/`, `build:all`, `pnpm`,
  `landing/`, or `VITE_DATASET`

## Success Criteria

- [x] `verify-parity.js` committed and exiting 0 on all four datasets
- [x] Parity result report committed under `plans/reports/`
- [x] Zero unexpected non-NULL columns
- [x] Manual checklist passes on the hub and all four dataset routes
- [x] `verify-parity.js` has zero npm dependencies (`node:sqlite` only)
- [x] `README.md` and `docs/` describe the actual repo, with npm commands
- [x] No stale path, pnpm, or build-variant references anywhere outside `plans/`

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Parity failure discovered late, after the data move | Phase 1 runs the same comparison in place before Phase 2 moves anything |
| Gate weakened to make it pass | Explicit step 3: a mismatch is a Phase 1 bug, not a gate-tuning problem |
| Docs merged by picking one copy and discarding the other | Step 6 requires reading both; they document different pipelines |
| Baseline missing because Phase 1 step 1 was skipped | Phase 1 treats it as blocking; without it this phase cannot run |
