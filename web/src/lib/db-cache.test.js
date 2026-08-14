import { describe, expect, it } from "vitest";
import { cacheKey, keepOnly, matchAny, matchVersion } from "./db-cache.js";

/**
 * Enough of the Cache Storage contract for these functions to run against.
 *
 * keys() hands back request-shaped objects and match() accepts them, because
 * that round trip — keys() then match(request) — is how matchAny works.
 */
function fakeCache(entries = {}) {
  const store = new Map(Object.entries(entries));
  const asUrl = (key) => (typeof key === "string" ? key : key.url);
  return {
    store,
    async match(key) {
      return store.get(asUrl(key)) ?? undefined;
    },
    async put(key, response) {
      store.set(asUrl(key), response);
    },
    async delete(key) {
      return store.delete(asUrl(key));
    },
    async keys() {
      return [...store.keys()].map((url) => ({ url }));
    },
  };
}

const URL_2016 = "/thptqg/db/2016.sqlite3";

describe("cacheKey", () => {
  it("versions the entry by ETag", () => {
    expect(cacheKey(URL_2016, '"abc123"')).toBe(`${URL_2016}?v=%22abc123%22`);
  });
});

describe("matchVersion", () => {
  it("returns the copy stored for that exact version", async () => {
    const cache = fakeCache({ [cacheKey(URL_2016, "v1")]: "body" });
    await expect(matchVersion(cache, URL_2016, "v1")).resolves.toBe("body");
  });

  it("misses when the server has a different version", async () => {
    const cache = fakeCache({ [cacheKey(URL_2016, "v1")]: "body" });
    await expect(matchVersion(cache, URL_2016, "v2")).resolves.toBeNull();
  });

  it("copes with no cache and no etag", async () => {
    await expect(matchVersion(null, URL_2016, "v1")).resolves.toBeNull();
    await expect(matchVersion(fakeCache(), URL_2016, null)).resolves.toBeNull();
  });
});

describe("matchAny", () => {
  it("finds a stored copy whatever its version, for when the server is unreachable", async () => {
    const cache = fakeCache({ [cacheKey(URL_2016, "old")]: "body" });
    await expect(matchAny(cache, URL_2016)).resolves.toBe("body");
  });

  it("does not return another dataset's database", async () => {
    const cache = fakeCache({ [cacheKey("/thptqg/db/2017.sqlite3", "v1")]: "body" });
    await expect(matchAny(cache, URL_2016)).resolves.toBeNull();
  });
});

describe("keepOnly", () => {
  it("stores the new version and drops the old one", async () => {
    const cache = fakeCache({ [cacheKey(URL_2016, "old")]: "stale" });

    await expect(keepOnly(cache, URL_2016, "new", "fresh")).resolves.toBe(true);

    expect([...cache.store.keys()]).toEqual([cacheKey(URL_2016, "new")]);
  });

  it("leaves the other dataset alone", async () => {
    const other = cacheKey("/thptqg/db/2017.sqlite3", "v1");
    const cache = fakeCache({ [other]: "keep me" });

    await keepOnly(cache, URL_2016, "new", "fresh");

    expect(cache.store.has(other)).toBe(true);
  });

  it("reports failure rather than throwing when the disk is full", async () => {
    const cache = fakeCache();
    cache.put = async () => {
      throw new Error("QuotaExceededError");
    };
    await expect(keepOnly(cache, URL_2016, "new", "fresh")).resolves.toBe(false);
  });
});
