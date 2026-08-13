#!/usr/bin/env node
/**
 * Assemble the GitHub Pages artifact from a single Vite build.
 *
 * The app resolves its dataset from the URL, so every page is the same
 * index.html. Because `base` is absolute (/thptqg/), that file references
 * /thptqg/assets/... no matter which directory it is served from — so copying
 * it to each dataset path produces a real static file at every URL.
 *
 * GitHub Pages serves those as directory indexes, which is why this needs no
 * SPA 404-fallback redirect. That matters beyond tidiness: the usual fallback
 * rewrites the URL and would interfere with the ?q= deep links the app relies
 * on.
 *
 * Usage: node web/scripts/assemble-site.js
 */

import { cpSync, mkdirSync, rmSync, existsSync, readdirSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { DATASET_IDS } from "../src/datasets.js";

// The build output belongs to this workspace; the Pages artifact does not. The
// deploy action uploads _site from the repository root, so that one climbs out.
const WEB = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const DIST = join(WEB, "dist");
const SITE = join(WEB, "..", "_site");

// The two legacy nested URLs (2017/old, 2017/old2) are gone with the datasets
// they addressed. Unknown paths render the hub via 404.html, so old bookmarks
// land somewhere useful rather than on a Pages error page.

if (!existsSync(join(DIST, "index.html"))) {
  console.error(`no build found at ${DIST} — run: npm run build`);
  process.exit(1);
}

rmSync(SITE, { recursive: true, force: true });
mkdirSync(SITE, { recursive: true });

// Base build: index.html, assets/, and the gzipped databases from publicDir.
cpSync(DIST, SITE, { recursive: true });

// Unknown paths render the hub rather than the default Pages 404.
cpSync(join(DIST, "index.html"), join(SITE, "404.html"));

// One entry point per dataset.
for (const path of DATASET_IDS) {
  mkdirSync(join(SITE, path), { recursive: true });
  cpSync(join(DIST, "index.html"), join(SITE, path, "index.html"));
}

// Only gzipped databases may ship. build-db.js gzips without -k so no raw .db
// should exist, but publicDir is copied wholesale — a leftover from an
// interrupted build would go straight through, and a raw database is 100+ MB.
// SQLite also leaves .db-journal files mid-build, so anything that is not a
// .gz is rejected rather than just files ending in .db.
const stray = [];
(function walk(dir) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) walk(full);
    else if (/\.db(-journal|-wal|-shm)?$/.test(entry)) stray.push(full);
  }
})(SITE);

if (stray.length) {
  console.error("uncompressed database artefact(s) found in the site output:");
  for (const f of stray) {
    console.error(`  ${f}  (${(statSync(f).size / 1048576).toFixed(1)} MB)`);
  }
  console.error("\nremove them from .build/public/db and re-run");
  process.exit(1);
}

console.log(`assembled ${SITE}`);
for (const path of ["", "404.html", ...DATASET_IDS]) {
  console.log(`  /thptqg/${path}`);
}
