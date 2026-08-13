/**
 * Path → dataset resolution.
 *
 * URLs are flat, one segment per dataset, and the segment IS the dataset id:
 *
 *   /thptqg/        → null (hub)
 *   /thptqg/2016/   → 2016
 *   /thptqg/2017/   → 2017
 *
 * That makes this an exact match on a single segment. The nested form this
 * replaced (/thptqg/2017/old/) needed longest-prefix matching, because
 * "/thptqg/2017/old/" also starts with "/thptqg/2017/" — an entire class of
 * routing bug the flat scheme designs out.
 *
 * Those nested URLs used to be rewritten here so old bookmarks kept working.
 * Both of them addressed the 2017-old and 2017-old2 datasets, which no longer
 * exist, so there is nothing left to rewrite them to: they now fall through to
 * the hub like any other unknown path.
 */

import { DATASETS } from "./datasets";

const BASE = import.meta.env.BASE_URL;

/** Strip the site base and any trailing slash, leaving a bare route segment. */
function toSegment(pathname) {
  const path = pathname.startsWith(BASE) ? pathname.slice(BASE.length) : pathname;
  return path.replace(/^\/+/, "").replace(/\/+$/, "");
}

/** Resolve the current location to a dataset, or null for the hub. */
export function resolveRoute(pathname = window.location.pathname) {
  const segment = toSegment(pathname);
  return DATASETS.find((d) => d.id === segment) ?? null;
}
