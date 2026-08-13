/**
 * The four datasets this site serves.
 *
 * `id` is the single identifier used end to end:
 *
 *   data/<id>/  →  parser/configs/<id>.toml  →  db/<id>.db.gz  →  /thptqg/<id>/
 *
 * Site path and database URL are derived from `id` rather than stored, so a
 * dataset cannot be misconfigured into pointing at the wrong database.
 *
 * Imported by the Vite app *and* by parser/scripts/build-db.js under plain
 * Node, so this module must stay free of `import.meta.env` and any Vite-only
 * syntax. Callers pass the base URL in explicitly for that reason.
 */

// Extension is required: this module is also imported by plain Node
// (parser/scripts/build-db.js), which does not resolve extensionless paths.
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
  {
    id: "2017-old",
    year: 2017,
    label: "2017 — bản cũ",
    blurb: "847.348 thí sinh · bản trước khi cập nhật",
    title: "Tra cứu điểm thi THPT Quốc gia 2017 (bản cũ)",
    subtitle: SUBTITLE,
    source: "bản lưu trước đợt cập nhật baotintuc.vn",
    dbSizeMb: 47,
    examples: ["49008235", "Nguyễn Minh Tiến"],
    presets: PRESETS_2017,
  },
  {
    id: "2017-old2",
    year: 2017,
    label: "2017 — bản cũ 2",
    blurb: "679.764 thí sinh · bản xuất lại",
    title: "Tra cứu điểm thi THPT Quốc gia 2017 (bản cũ 2)",
    subtitle: SUBTITLE,
    source: "bản xuất lại đã hiệu chỉnh",
    dbSizeMb: 38,
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
