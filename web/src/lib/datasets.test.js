import { describe, expect, it } from "vitest";
import { DATASETS, dbOf, pathOf } from "./datasets.js";

const [dataset] = DATASETS;

describe("dataset locations", () => {
  it("derives the site path from the id", () => {
    expect(pathOf(dataset, "/thptqg")).toBe(`/thptqg/${dataset.id}/`);
  });

  it("names the database file the assembler publishes", () => {
    expect(dbOf(dataset, "/thptqg")).toBe(`/thptqg/db/${dataset.id}.sqlite3`);
  });
});
