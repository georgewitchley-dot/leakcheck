import { describe, expect, it } from "vitest";
import { findCount, sha1Hex } from "./pwned";

describe("sha1Hex", () => {
  it("hashes 'password' to the known SHA-1", async () => {
    expect(await sha1Hex("password")).toBe("5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8");
  });
  it("handles unicode", async () => {
    expect(await sha1Hex("pässwörd")).toHaveLength(40);
  });
});

describe("findCount", () => {
  const body = "0018A45C4D1DEF81644B54AB7F969B88D65:0\r\n1E4C9B93F3F0682250B6CF8331B7EE68FD8:10434004\r\n011053FD0102E94D6AE2F8B83D76FAF94F6:3";
  it("finds a matching suffix", () => {
    expect(findCount(body, "1E4C9B93F3F0682250B6CF8331B7EE68FD8")).toBe(10434004);
  });
  it("returns 0 for padding and misses", () => {
    expect(findCount(body, "0018A45C4D1DEF81644B54AB7F969B88D65")).toBe(0);
    expect(findCount(body, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")).toBe(0);
  });
});
