import { describe, expect, it } from "vitest";
import { DATASETS, dbOf, dbPrefixOf, dbSourceOf } from "./datasets.js";

const [dataset] = DATASETS;

/**
 * sql.js-httpvfs builds every request URL as urlPrefix + chunk index, and one
 * chunk holds the whole database, so the index is always 0. If these two ever
 * stop agreeing, the library asks for a file the assembler never published and
 * every query 404s — which no other test would catch.
 */
describe("database location", () => {
  it("puts the chunk index where the published file name ends", () => {
    expect(dbOf(dataset, "/thptqg")).toBe(`${dbPrefixOf(dataset, "/thptqg")}0`);
  });

  it("names the file the assembler publishes", () => {
    expect(dbOf(dataset, "/thptqg")).toBe(`/thptqg/db/${dataset.id}.sqlite30`);
  });

  it("hands RemoteDatabase both forms of the same location", () => {
    const source = dbSourceOf(dataset, "/thptqg");
    expect(source).toEqual({
      url: dbOf(dataset, "/thptqg"),
      urlPrefix: dbPrefixOf(dataset, "/thptqg"),
    });
  });
});
