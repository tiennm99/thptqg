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

import registry from "../../../datasets.json";

import { PRESETS_2016, PRESETS_2017 } from "./sql-presets";

const SUBTITLE = "Dữ liệu thí sinh toàn quốc · Hỗ trợ truy vấn SQL tùy chỉnh";

/**
 * Presentation, keyed by the ids declared in datasets.json.
 *
 * `source` is the full article URL the dataset's spreadsheets come from. The
 * canonical copy is the crawler's `Article` field
 * (crawler/internal/sources/source_<id>.go); it is duplicated here because a Go
 * module and the web app cannot share a constant. Keep the two in step.
 *
 * `sourceName` is who published that article, and is what the footer shows —
 * the URLs run to 120 characters and used to wrap across three lines on a
 * phone. The link still points at the article, and its title attribute still
 * carries the URL for anyone who wants to see where it goes.
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
    sourceName: "Trường Phổ thông DTNT tỉnh Bắc Ninh",
    examples: ["TKG002747", "Nguyễn Bửu Lộc"],
    presets: PRESETS_2016,
  },
  2017: {
    label: "Kỳ thi 2017",
    title: "Tra cứu điểm thi THPT Quốc gia 2017",
    subtitle: SUBTITLE,
    source:
      "https://baotintuc.vn/tuyen-sinh/tra-cuu-diem-thi-thpt-2017-cua-63-tinh-thanh-pho-tren-baotintucvn-20170706073512672.htm",
    sourceName: "Báo Tin tức và Dân tộc - TTXVN",
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
  // The footer renders sourceName as the link text, so a missing one is an
  // empty link rather than a visible mistake.
  for (const field of ["title", "label", "source", "sourceName", "examples", "presets"]) {
    if (!CONTENT[id][field]) {
      throw new Error(`web/src/lib/datasets.js: dataset "${id}" is missing ${field}`);
    }
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
  rows: d.expectedRows,
  // Derived from expectedRows so the count the hub shows is the same number the
  // assembler enforces.
  blurb: `${d.expectedRows.toLocaleString("vi-VN")} thí sinh`,
  ...CONTENT[d.id],
}));

/** Look up a dataset by the route segment, or undefined for an unknown id. */
export function datasetById(id) {
  return DATASETS.find((d) => d.id === id);
}

/**
 * Site path for a dataset. `base` is SvelteKit's, which carries no trailing
 * slash: pathOf(d, "/thptqg") → "/thptqg/2017/".
 */
export function pathOf(dataset, base) {
  return `${base}/${dataset.id}/`;
}

/**
 * The chunk the whole database lives in. One chunk covers the file, so this is
 * the only index the library ever asks for, and the assembler publishes the
 * database under a name ending in it.
 */
const CHUNK_INDEX = "0";

/**
 * Everything a database URL has except the chunk index, e.g.
 * dbPrefixOf(d, "/thptqg") → "/thptqg/db/2017.sqlite3".
 *
 * sql.js-httpvfs reads the file in chunked mode and builds each URL as
 * prefix + chunk index. One chunk holds the whole database, so the index is
 * always 0 and the published file is "<id>.sqlite30".
 */
export function dbPrefixOf(dataset, base) {
  return `${base}/db/${dataset.id}.sqlite3`;
}

/**
 * The published database file, e.g. dbOf(d, "/thptqg") → "/thptqg/db/2017.sqlite30".
 *
 * Uncompressed on purpose: the browser reads byte ranges of it, and a range of
 * a gzip stream is not a range of the database.
 */
export function dbOf(dataset, base) {
  return `${dbPrefixOf(dataset, base)}${CHUNK_INDEX}`;
}

/**
 * Both forms of the database's location, for RemoteDatabase: the file to read,
 * and the prefix the library appends the chunk index to.
 */
export function dbSourceOf(dataset, base) {
  return { url: dbOf(dataset, base), urlPrefix: dbPrefixOf(dataset, base) };
}
