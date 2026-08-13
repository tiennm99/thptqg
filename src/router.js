/**
 * Path → dataset resolution.
 *
 * URLs are flat, one segment per dataset, and the segment IS the dataset id:
 *
 *   /thptqg/            → null (hub)
 *   /thptqg/2016/       → 2016
 *   /thptqg/2017-old/   → 2017-old
 *
 * That makes this an exact match on a single segment. The nested form this
 * replaced (/thptqg/2017/old/) needed longest-prefix matching, because
 * "/thptqg/2017/old/" also starts with "/thptqg/2017/" — an entire class of
 * routing bug the flat scheme designs out.
 */

import { DATASETS } from "./datasets";

/**
 * URLs published before the switch to flat paths. Kept working so existing
 * links and bookmarks resolve instead of 404ing.
 */
const LEGACY_PATHS = {
  "2017/old": "2017-old",
  "2017/old2": "2017-old2",
};

const BASE = import.meta.env.BASE_URL;

/** Strip the site base and any trailing slash, leaving a bare route segment. */
function toSegment(pathname) {
  const path = pathname.startsWith(BASE) ? pathname.slice(BASE.length) : pathname;
  return path.replace(/^\/+/, "").replace(/\/+$/, "");
}

/**
 * Resolve the current location to a dataset, or null for the hub.
 *
 * Rewrites legacy nested URLs to their flat equivalent, preserving the query
 * string — a `?q=` deep link must survive the redirect, which is exactly what
 * the usual SPA 404-fallback hack would break.
 */
export function resolveRoute(pathname = window.location.pathname) {
  let segment = toSegment(pathname);

  const flat = LEGACY_PATHS[segment];
  if (flat) {
    window.history.replaceState(
      {},
      "",
      `${BASE}${flat}/${window.location.search}${window.location.hash}`,
    );
    segment = flat;
  }

  return DATASETS.find((d) => d.id === segment) ?? null;
}
