/**
 * The datasets this site serves.
 *
 * Which datasets exist is decided by the repository-root `datasets.json`, not
 * here: the assembler is a Go program and cannot import this module. This file
 * supplies only what the interface needs — titles, labels, search examples and
 * SQL presets, keyed by id.
 *
 * Site path and database URL are derived from `id` rather than stored, so a
 * dataset cannot be misconfigured into pointing at the wrong database.
 */

import registry from "../../datasets.json";

import { PRESETS_2016, PRESETS_2017 } from "./lib/sql-presets.js";

const SUBTITLE = "Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh";

/**
 * Presentation, keyed by the ids declared in datasets.json.
 *
 * `source` is the full article URL the dataset's spreadsheets come from, shown
 * in the footer as a link. The canonical copy is the crawler's `Article` field
 * (crawler/internal/sources/source_<id>.go); it is duplicated here because a Go
 * module and a Vite app cannot share a constant. Keep the two in step.
 */
const CONTENT = {
  2016: {
    label: "Kỳ thi 2016",
    title: "Tra cứu điểm thi THPT Quốc gia 2016",
    subtitle: SUBTITLE,
    // A school site that aggregated every exam cluster's spreadsheet, not the
    // ministry — hence the unexpected domain.
    source:
      "https://dtnt.bacninh.edu.vn/tin-tuc/tin-tuc-su-kien/cong-bo-diem-thi-thptqg-2016-toan-bo-120-cum-thi-da-co-diem.html",
    examples: ["TKG002747", "Nguyễn Bửu Lộc"],
    presets: PRESETS_2016,
  },
  2017: {
    label: "Kỳ thi 2017",
    title: "Tra cứu điểm thi THPT Quốc gia 2017",
    subtitle: SUBTITLE,
    source:
      "https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm",
    examples: ["49008235", "Nguyễn Minh Tiến"],
    presets: PRESETS_2017,
  },
};

// Fail at module load rather than in the browser: a registry entry with no
// content here renders a page with no title and no presets, and content for an
// unregistered id links to a database that was never built.
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
  // Derived from expectedRows so the count the hub shows is the same number the
  // assembler enforces.
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
