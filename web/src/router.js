/**
 * Path → dataset resolution.
 *
 * URLs are flat, one segment per dataset, and the segment IS the dataset id:
 *
 *   /thptqg/        → null (hub)
 *   /thptqg/2016/   → 2016
 *   /thptqg/2017/   → 2017
 *
 * Keeping them flat makes this an exact match on a single segment; nested
 * routes would need longest-prefix matching to disambiguate. Unknown paths fall
 * through to the hub.
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
