import { describe, expect, it } from "vitest";
import { MIN_NAME_CHARS, MIN_SBD_DIGITS, detectMode, isExamId, normaliseExamId } from "./query-mode";

describe("isExamId", () => {
  it("accepts every exam-number shape in the corpus", () => {
    expect(isExamId("49008235")).toBe(true); // 2017, 8 digits
    expect(isExamId("017006021")).toBe(true); // 2016, 9 digits with leading zero
    expect(isExamId("BAL000001")).toBe(true); // 2016, cluster prefix
    expect(isExamId("TKG002747")).toBe(true);
    expect(isExamId("bal000001")).toBe(true); // lower case is normalised later
  });

  it("rejects names and prefixes longer than a cluster code", () => {
    expect(isExamId("Nguyen Van A")).toBe(false);
    expect(isExamId("NGUYEN01")).toBe(false); // 6 letters, not a cluster code
    expect(isExamId("")).toBe(false);
    expect(isExamId("2016-01")).toBe(false);
  });
});

describe("normaliseExamId", () => {
  it("upper-cases the cluster prefix and trims", () => {
    expect(normaliseExamId("  bal000001 ")).toBe("BAL000001");
    expect(normaliseExamId("49008235")).toBe("49008235");
  });
});

describe("detectMode", () => {
  it("classifies an empty box", () => {
    expect(detectMode("   ").mode).toBe("empty");
  });

  it("classifies exam numbers, short ones separately", () => {
    expect(detectMode("49008235").mode).toBe("sbd");
    expect(detectMode("1".repeat(MIN_SBD_DIGITS)).mode).toBe("sbd");
    expect(detectMode("1".repeat(MIN_SBD_DIGITS - 1)).mode).toBe("sbd-short");
  });

  it("classifies names, short ones separately", () => {
    expect(detectMode("Nguyễn").mode).toBe("name");
    expect(detectMode("a".repeat(MIN_NAME_CHARS)).mode).toBe("name");
    expect(detectMode("a".repeat(MIN_NAME_CHARS - 1)).mode).toBe("name-short");
  });

  it("always returns a hint to show under the field", () => {
    for (const q of ["", "49008235", "1", "Nguyễn Bửu Lộc", "a"]) {
      expect(detectMode(q).hint.length).toBeGreaterThan(0);
    }
  });
});
