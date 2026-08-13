/**
 * The datasets this site serves.
 *
 * `id` is the single identifier used end to end:
 *
 *   data/<id>/  →  parser/configs/<id>.yml  →  db/<id>.db.gz  →  /thptqg/<id>/
 *
 * Site path and database URL are derived from `id` rather than stored, so a
 * dataset cannot be misconfigured into pointing at the wrong database.
 *
 * Which datasets exist is not decided here. That lives in the repository-root
 * `datasets.json`, which the assembler reads too — it is a Go program and
 * cannot import this module. This file supplies only what the interface needs:
 * titles, labels, search examples and SQL presets, keyed by id.
 */

import registry from "../../datasets.json";

import { PRESETS_2016, PRESETS_2017 } from "./lib/sql-presets.js";

const SUBTITLE = "Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh";

/** Presentation, keyed by the ids declared in datasets.json. */
const CONTENT = {
  2016: {
    label: "Kỳ thi 2016",
    title: "Tra cứu điểm thi THPT Quốc gia 2016",
    subtitle: SUBTITLE,
    source: "Bộ GD&ĐT",
    examples: ["17006021", "Nguyễn Thị Hoa"],
    presets: PRESETS_2016,
  },
  2017: {
    label: "Kỳ thi 2017",
    title: "Tra cứu điểm thi THPT Quốc gia 2017",
    subtitle: SUBTITLE,
    source: "baotintuc.vn",
    examples: ["49008235", "Nguyễn Minh Tiến"],
    presets: PRESETS_2017,
  },
};

// A dataset in the registry with no content here would render a page with no
// title and no presets; content here for a dataset that no longer exists would
// offer a link to a database that was never built. Neither should reach a
// browser, so both fail at module load.
for (const { id } of registry.datasets) {
  if (!CONTENT[id]) {
    throw new Error(`datasets.json declares "${id}" but web/src/datasets.js has no content for it`);
  }
}
for (const id of Object.keys(CONTENT)) {
  if (!registry.datasets.some((d) => d.id === id)) {
    throw new Error(`web/src/datasets.js has content for "${id}", which datasets.json does not declare`);
  }
}

export const DATASETS = registry.datasets.map((d) => ({
  id: d.id,
  dbSizeMb: d.dbSizeMb,
  // Derived, not written twice: the candidate count the hub shows is the same
  // number the assembler enforces, so the two cannot drift apart.
  blurb: `${d.expectedRows.toLocaleString("vi-VN")} thí sinh`,
  ...CONTENT[d.id],
}));

/** Site path for a dataset, e.g. pathOf(d, "/thptqg/") → "/thptqg/2017/". */
export function pathOf(dataset, base) {
  return `${base}${dataset.id}/`;
}

/** Gzipped database URL, e.g. dbOf(d, "/thptqg/") → "/thptqg/db/2017.db.gz". */
export function dbOf(dataset, base) {
  return `${base}db/${dataset.id}.db.gz`;
}
