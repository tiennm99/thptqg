/**
 * The four datasets this site serves.
 *
 * `id` is the single identifier used end to end:
 *
 *   data/<id>/  →  go-parser/configs/<id>.yml  →  db/<id>.db.gz  →  /thptqg/<id>/
 *
 * Site path and database URL are derived from `id` rather than stored, so a
 * dataset cannot be misconfigured into pointing at the wrong database.
 *
 * Imported by the Vite app *and* by go-parser/scripts/build-db.js under plain
 * Node, so this module must stay free of `import.meta.env` and any Vite-only
 * syntax. Callers pass the base URL in explicitly for that reason.
 *
 * That cross-package import is why this file cannot move into a Vite-only
 * corner of the app: go-parser reads DATASET_IDS and dbSizeMb straight from
 * here, and reaches across the workspace boundary to do it.
 */

// Extension is required: this module is also imported by plain Node
// (go-parser/scripts/build-db.js), which does not resolve extensionless paths.
import { PRESETS_2016, PRESETS_2017 } from "./lib/sql-presets.js";

const SUBTITLE = "Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh";

export const DATASETS = [
  {
    id: "2016",
    year: 2016,
    label: "Kỳ thi 2016",
    blurb: "877.461 thí sinh",
    title: "Tra cứu điểm thi THPT Quốc gia 2016",
    subtitle: SUBTITLE,
    source: "Bộ GD&ĐT",
    dbSizeMb: 44,
    examples: ["17006021", "Nguyễn Thị Hoa"],
    presets: PRESETS_2016,
  },
  {
    id: "2017",
    year: 2017,
    label: "Kỳ thi 2017",
    blurb: "861.068 thí sinh",
    title: "Tra cứu điểm thi THPT Quốc gia 2017",
    subtitle: SUBTITLE,
    source: "baotintuc.vn",
    dbSizeMb: 48,
    examples: ["49008235", "Nguyễn Minh Tiến"],
    presets: PRESETS_2017,
  },
];

/** Dataset IDs in build order. */
export const DATASET_IDS = DATASETS.map((d) => d.id);

/** Site path for a dataset, e.g. pathOf(d, "/thptqg/") → "/thptqg/2017-old/". */
export function pathOf(dataset, base) {
  return `${base}${dataset.id}/`;
}

/** Gzipped database URL, e.g. dbOf(d, "/thptqg/") → "/thptqg/db/2017-old.db.gz". */
export function dbOf(dataset, base) {
  return `${base}db/${dataset.id}.db.gz`;
}
