import { describe, expect, it } from "vitest";
import { ADMISSION_BLOCKS, computeBlocks, scoreTier } from "./admission-blocks";

/** A row with every column NULL, so a test only states the scores it needs. */
function student(scores) {
  return {
    so_bao_danh: "49008235",
    ho_ten: "Nguyễn Văn A",
    ho_ten_ascii: "nguyen van a",
    ngay_sinh: null,
    ten_cum_thi: null,
    gioi_tinh: null,
    toan: null,
    ngu_van: null,
    vat_ly: null,
    hoa_hoc: null,
    sinh_hoc: null,
    khtn: null,
    lich_su: null,
    dia_ly: null,
    gdcd: null,
    khxh: null,
    tieng_anh: null,
    tieng_phap: null,
    tieng_nga: null,
    tieng_duc: null,
    tieng_nhat: null,
    tieng_trung: null,
    ...scores,
  };
}

describe("scoreTier", () => {
  it("returns nothing for a subject the candidate did not sit", () => {
    expect(scoreTier(null)).toBeNull();
    expect(scoreTier(undefined)).toBeNull();
  });

  it("treats điểm liệt as an inclusive bound and the rest as exclusive", () => {
    // ≤ 1 is an automatic fail whatever the other scores are.
    expect(scoreTier(0)?.key).toBe("common");
    expect(scoreTier(1)?.key).toBe("common");
    expect(scoreTier(1.01)?.key).toBe("uncommon");
    expect(scoreTier(4.99)?.key).toBe("uncommon");
    expect(scoreTier(5)?.key).toBe("rare");
    expect(scoreTier(6.5)?.key).toBe("epic");
    expect(scoreTier(8)?.key).toBe("legendary");
    expect(scoreTier(9)?.key).toBe("prismatic");
    expect(scoreTier(10)?.key).toBe("prismatic");
  });

  it("carries a symbol as well as a colour, so meaning is never colour-only", () => {
    for (const score of [0, 3, 6, 7, 8.5, 9.5]) {
      expect(scoreTier(score)?.symbol).toBeTruthy();
    }
  });
});

describe("computeBlocks", () => {
  it("skips a block when any of its three subjects is missing", () => {
    const blocks = computeBlocks(student({ toan: 8, vat_ly: 7 }));
    expect(blocks.find((b) => b.code === "A00")).toBeUndefined();
  });

  it("sums exactly three subjects and sorts by total desc", () => {
    const blocks = computeBlocks(student({ toan: 8, vat_ly: 7, hoa_hoc: 6, tieng_anh: 9 }));
    const a00 = blocks.find((b) => b.code === "A00");
    const a01 = blocks.find((b) => b.code === "A01");
    expect(a00?.total).toBe(21);
    expect(a01?.total).toBe(24);
    expect(a00?.parts).toHaveLength(3);
    expect(blocks[0].total).toBeGreaterThanOrEqual(blocks[blocks.length - 1].total);
  });

  it("self-excludes 2017-only blocks on a 2016 row, with no per-year branching", () => {
    // GDCD did not exist in 2016, so every block needing it drops out.
    const blocks = computeBlocks(student({ toan: 8, lich_su: 7, dia_ly: 6 }));
    expect(blocks.map((b) => b.code)).toContain("A07");
    expect(blocks.map((b) => b.code)).not.toContain("A08");
  });

  it("keeps every block a 3-subject sum", () => {
    for (const block of ADMISSION_BLOCKS) {
      expect(block.subjects).toHaveLength(3);
    }
  });
});
