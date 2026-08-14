import { describe, expect, it } from "vitest";
import { isAsciiOnly, toAscii } from "./to-ascii";

/*
 * These pairs are the contract with the Go parser: `ho_ten_ascii` is written by
 * ToAscii in parser/internal/transform/transform.go, and searched by toAscii
 * here. The two implementations must agree character for character or a name
 * search silently misses rows. The cases below are the ones
 * parser/internal/transform/transform_test.go pins on the Go side.
 */
const GO_PARITY = [
  ["Nguyễn Bửu Lộc", "nguyen buu loc"],
  ["NGUYỄN THỊ HOA", "nguyen thi hoa"],
  ["Trần Thị Phước An", "tran thi phuoc an"],
  ["Đặng Văn Đức", "dang van duc"],
  ["ĐINH TIÊN HOÀNG", "dinh tien hoang"],
  ["Lê Thị Thúy ái", "le thi thuy ai"],
  ["Hồ Diệu á", "ho dieu a"],
  ["", ""],
];

describe("toAscii", () => {
  it.each(GO_PARITY)("folds %j to %j", (input, want) => {
    expect(toAscii(input)).toBe(want);
  });

  it("strips only U+0300..U+036F, not every combining mark", () => {
    // U+0654 (Arabic hamza above) is a combining mark outside the dropped
    // range, so it survives. This is the same pair the Go side pins in
    // TestToAsciiUsesLiteralRangeNotUnicodeMn; widening either implementation
    // to "every combining mark" breaks it.
    expect(toAscii("a\u0654b")).toBe("a\u0654b");
    expect(toAscii("\u00e1b")).toBe("ab");
  });

  it("handles đ regardless of case and of NFD decomposition", () => {
    expect(toAscii("Đ")).toBe("d");
    expect(toAscii("đ")).toBe("d");
  });
});

describe("isAsciiOnly", () => {
  it("separates plain queries from accented ones", () => {
    expect(isAsciiOnly("nguyen van a")).toBe(true);
    expect(isAsciiOnly("47006585")).toBe(true);
    expect(isAsciiOnly("Nguyễn")).toBe(false);
  });
});
