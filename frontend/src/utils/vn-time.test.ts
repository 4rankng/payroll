import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formatVnTime, isVnNextDay, parseEpochMs, VN_TIMEZONE } from "./vn-time";

// Reproduces the production bug (ROOT CAUSE of "Chưa đến giờ vào làm" at 20:17 VN
// for a 20:00 shift): the previous window-gate compared getHours() (device-local)
// against server-supplied HH:mm Vietnam strings, so any device not set to
// Asia/Ho_Chi_Minh was mis-gated. These tests pin the timezone-independent
// behavior of the helpers that replaced that logic.

describe("parseEpochMs", () => {
  it("parses a +07:00 RFC 3339 instant to epoch ms", () => {
    // 2026-07-12T20:00:00+07:00 == 2026-07-12T13:00:00Z
    expect(parseEpochMs("2026-07-12T20:00:00+07:00")).toBe(Date.parse("2026-07-12T13:00:00Z"));
  });

  it("parses a Z instant", () => {
    expect(parseEpochMs("2026-07-12T13:00:00.000Z")).toBe(Date.parse("2026-07-12T13:00:00Z"));
  });

  it("returns null for empty/invalid input", () => {
    expect(parseEpochMs(undefined)).toBeNull();
    expect(parseEpochMs(null)).toBeNull();
    expect(parseEpochMs("")).toBeNull();
    expect(parseEpochMs("not-a-date")).toBeNull();
  });
});

describe("formatVnTime", () => {
  // Force a deterministic non-VN system timezone to prove the helper does NOT
  // depend on the host timezone. Asia/Ho_Chi_Minh is UTC+7; setting the host
  // to UTC means a naive getHours() would read 13 where the VN hour is 20.
  const originalTZ = process.env.TZ;
  beforeEach(() => {
    process.env.TZ = "UTC";
    // Intl formatters pick up TZ changes lazily; date-fns/Date constructors also
    // rely on it. No extra reset is needed for Intl in modern V8.
  });
  afterEach(() => {
    process.env.TZ = originalTZ;
  });

  it("formats a +07:00 instant as the Vietnam HH:mm (not the host-local hour)", () => {
    // Even with the host in UTC, this must read as 20:00 (VN), not 13:00 (UTC).
    expect(formatVnTime("2026-07-12T20:00:00+07:00")).toBe("20:00");
  });

  it("formats a Z instant as the Vietnam HH:mm", () => {
    // 13:17Z == 20:17 +07:00 — the exact moment the employee in the bug report
    // tried to check in.
    expect(formatVnTime("2026-07-12T13:17:00.000Z")).toBe("20:17");
  });

  it("formats a midnight-VN instant as 00:xx, not 24:xx", () => {
    expect(formatVnTime("2026-07-13T00:00:00+07:00")).toBe("00:00");
  });

  it("returns empty string for falsy/invalid input", () => {
    expect(formatVnTime(undefined)).toBe("");
    expect(formatVnTime("")).toBe("");
    expect(formatVnTime("garbage")).toBe("");
  });

  it("VN_TIMEZONE constant is Asia/Ho_Chi_Minh", () => {
    expect(VN_TIMEZONE).toBe("Asia/Ho_Chi_Minh");
  });
});

describe("isVnNextDay", () => {
  it("flags a cross-midnight shift end", () => {
    // 22:00 -> 06:00 next day
    expect(isVnNextDay("2026-07-12T22:00:00+07:00", "2026-07-13T06:00:00+07:00")).toBe(true);
  });

  it("does not flag a same-day shift end", () => {
    expect(isVnNextDay("2026-07-12T08:00:00+07:00", "2026-07-12T17:00:00+07:00")).toBe(false);
  });

  it("handles falsy input safely", () => {
    expect(isVnNextDay(undefined, "2026-07-13T06:00:00+07:00")).toBe(false);
    expect(isVnNextDay("2026-07-12T22:00:00+07:00", "")).toBe(false);
  });
});
