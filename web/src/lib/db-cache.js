/**
 * Where a downloaded database is kept between visits.
 *
 * Cache Storage rather than IndexedDB: the thing being stored is an HTTP
 * response, the browser can stream it to disk instead of holding it as one
 * buffer, and eviction under storage pressure is the browser's business rather
 * than ours.
 *
 * Every entry is versioned by the server's ETag, because each deploy rebuilds
 * the databases and a stale copy would quietly answer with last week's data.
 * A new version is stored and the old ones for that database are dropped, so
 * one dataset never occupies the disk twice.
 */

const CACHE_NAME = "thptqg-db-v1";

/** The bucket, or null where Cache Storage is unavailable — SSR, or plain http. */
export async function openCache() {
  if (typeof caches === "undefined") return null;
  try {
    return await caches.open(CACHE_NAME);
  } catch {
    return null;
  }
}

/** Key for one version of one database. */
export function cacheKey(url, etag) {
  return `${url}?v=${encodeURIComponent(etag)}`;
}

/** The stored copy of exactly this version, or null. */
export async function matchVersion(cache, url, etag) {
  if (!cache || !etag) return null;
  return (await cache.match(cacheKey(url, etag))) ?? null;
}

/**
 * Any stored copy of this database, whichever version.
 *
 * The fallback for when the server cannot be reached at all: a slightly old
 * copy of frozen exam results is worth more than an error page.
 */
export async function matchAny(cache, url) {
  if (!cache) return null;
  const prefix = `${url}?v=`;
  for (const request of await cache.keys()) {
    if (request.url.includes(prefix)) return (await cache.match(request)) ?? null;
  }
  return null;
}

/**
 * Store this version, then drop every other version of the same database.
 *
 * Best effort: a full disk means the visitor downloads again next time, which
 * is worse than caching but no reason to fail the page they are on.
 */
export async function keepOnly(cache, url, etag, response) {
  if (!cache || !etag) return false;
  const key = cacheKey(url, etag);
  try {
    await cache.put(key, response);
  } catch {
    return false;
  }
  // Compared against the whole key, not the ETag alone: cache.keys() reports
  // absolute URLs while the key is a path, and two ETags can share a suffix.
  const prefix = `${url}?v=`;
  for (const request of await cache.keys()) {
    if (request.url.includes(prefix) && !request.url.endsWith(key)) {
      await cache.delete(request);
    }
  }
  return true;
}

/** Forget every stored database. */
export async function forgetAll() {
  if (typeof caches === "undefined") return;
  try {
    await caches.delete(CACHE_NAME);
  } catch {
    // Nothing to do: the copy stays until the browser evicts it.
  }
}
