/**
 * The four datasets this site serves.
 *
 * `id` is the single identifier used end to end:
 *
 *   data/<id>/  →  parser/configs/<id>.toml  →  db/<id>.db.gz  →  /thptqg/<id>/
 *
 * Imported by the Vite app *and* by parser/scripts/build-db.js under plain
 * Node, so this module must stay free of `import.meta.env` and any Vite-only
 * syntax. Callers pass the base URL in explicitly for that reason.
 */

export const DATASETS = [
  {
    id: "2016",
    year: 2016,
    label: "Kỳ thi 2016",
  },
  {
    id: "2017",
    year: 2017,
    label: "Kỳ thi 2017",
  },
  {
    id: "2017-old",
    year: 2017,
    label: "2017 — bản cũ",
  },
  {
    id: "2017-old2",
    year: 2017,
    label: "2017 — bản cũ 2",
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
