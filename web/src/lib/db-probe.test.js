import { describe, expect, it, vi } from "vitest";
import { looksLikeSqlite, parseTotalBytes, probeDatabase, readPageSize } from "./db-probe.js";

/** The first bytes of a real database: magic, NUL, then the page size at 16. */
function header(pageSize = 1024, magic = "SQLite format 3") {
  const bytes = new Uint8Array(100);
  for (let i = 0; i < magic.length; i += 1) bytes[i] = magic.charCodeAt(i);
  bytes[16] = pageSize >> 8;
  bytes[17] = pageSize & 0xff;
  return bytes;
}

function respond(bytes, { status = 206, contentRange = `bytes 0-99/${317096960}` } = {}) {
  return {
    status,
    headers: { get: (name) => (name.toLowerCase() === "content-range" ? contentRange : null) },
    arrayBuffer: async () => bytes.buffer,
  };
}

describe("parseTotalBytes", () => {
  it("takes the total from a Content-Range", () => {
    expect(parseTotalBytes("bytes 0-99/317096960")).toBe(317096960);
  });

  it("refuses an unknown total", () => {
    expect(() => parseTotalBytes("bytes 0-99/*")).toThrow(/how large/);
    expect(() => parseTotalBytes(null)).toThrow(/absent/);
  });
});

describe("readPageSize", () => {
  it("reads the two big-endian bytes at offset 16", () => {
    expect(readPageSize(header(1024))).toBe(1024);
    expect(readPageSize(header(4096))).toBe(4096);
  });

  it("treats 1 as 65536, as the file format does", () => {
    expect(readPageSize(header(1))).toBe(65536);
  });
});

describe("looksLikeSqlite", () => {
  it("accepts a real header", () => {
    expect(looksLikeSqlite(header())).toBe(true);
  });

  it("rejects a gzip stream, which is what a compressing host returns", () => {
    const gzip = new Uint8Array([0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00]);
    expect(looksLikeSqlite(gzip)).toBe(false);
  });

  it("requires the NUL that terminates the magic", () => {
    expect(looksLikeSqlite(header(1024, "SQLite format 3x"))).toBe(false);
  });
});

describe("probeDatabase", () => {
  it("asks for the header by range and returns the total length", async () => {
    const fetchImpl = vi.fn(async () => respond(header()));
    const total = await probeDatabase("/db/2016.sqlite30", 1024, fetchImpl);

    expect(total).toBe(317096960);
    expect(fetchImpl).toHaveBeenCalledWith("/db/2016.sqlite30", {
      headers: { Range: "bytes=0-99" },
    });
  });

  it("fails when the host ignores the range", async () => {
    const fetchImpl = async () => respond(header(), { status: 200 });
    await expect(probeDatabase("/db/2016.sqlite30", 1024, fetchImpl)).rejects.toThrow(/expected 206/);
  });

  it("fails when the bytes are not a database", async () => {
    const gzip = new Uint8Array([0x1f, 0x8b, 0x08]);
    const fetchImpl = async () => respond(gzip);
    await expect(probeDatabase("/db/2016.sqlite30", 1024, fetchImpl)).rejects.toThrow(
      /not a SQLite header/,
    );
  });

  it("warns, but continues, when the page size is not the request size", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const fetchImpl = async () => respond(header(4096));

    await expect(probeDatabase("/db/2016.sqlite30", 1024, fetchImpl)).resolves.toBe(317096960);
    expect(warn).toHaveBeenCalledWith(expect.stringMatching(/page size 4096/));
    warn.mockRestore();
  });
});
