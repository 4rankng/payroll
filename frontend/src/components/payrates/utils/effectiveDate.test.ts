import { describe, expect, it } from "vitest";
import { defaultCreateEffectiveFrom, earliestAllowedEffectiveFrom } from "./effectiveDate";

// Fixed 'today' so the tests are deterministic: 2026-11-18.
const TODAY = new Date("2026-11-18T00:00:00");

describe("earliestAllowedEffectiveFrom", () => {
  it("returns undefined when no config exists yet", () => {
    expect(earliestAllowedEffectiveFrom(null)).toBeUndefined();
    expect(earliestAllowedEffectiveFrom(undefined)).toBeUndefined();
  });

  it("floors at latest config + 1 day", () => {
    expect(earliestAllowedEffectiveFrom({ fromDate: "2025-12-09" })).toBe("2025-12-10");
  });

  it("uses the paid floor when it is later than latest + 1", () => {
    expect(
      earliestAllowedEffectiveFrom({ fromDate: "2025-12-09", earliest_effective_from: "2026-11-15" }),
    ).toBe("2026-11-15");
  });

  it("keeps latest + 1 when the paid floor is earlier", () => {
    expect(
      earliestAllowedEffectiveFrom({ fromDate: "2025-12-09", earliest_effective_from: "2025-12-01" }),
    ).toBe("2025-12-10");
  });
});

describe("defaultCreateEffectiveFrom", () => {
  it("defaults to the start of the month for a first-ever config", () => {
    expect(defaultCreateEffectiveFrom(null, TODAY)).toBe("2026-11-01");
    expect(defaultCreateEffectiveFrom(undefined, TODAY)).toBe("2026-11-01");
    expect(defaultCreateEffectiveFrom(null, new Date("2026-01-15T00:00:00"))).toBe("2026-01-01");
  });

  it("defaults to today — never to the past latest + 1 (PAYR-39)", () => {
    // Latest config started 09/12/2025; the old default was 10/12/2025,
    // which customers read as re-pricing history.
    expect(defaultCreateEffectiveFrom({ fromDate: "2025-12-09" }, TODAY)).toBe("2026-11-18");
    expect(
      defaultCreateEffectiveFrom({ fromDate: "2025-12-09", earliest_effective_from: "2025-12-01" }, TODAY),
    ).toBe("2026-11-18");
  });

  it("respects the paid floor when it is later than today", () => {
    expect(
      defaultCreateEffectiveFrom({ fromDate: "2025-12-09", earliest_effective_from: "2026-11-20" }, TODAY),
    ).toBe("2026-11-20");
  });

  it("respects a future latest + 1 (config already effective tomorrow or later)", () => {
    expect(defaultCreateEffectiveFrom({ fromDate: "2026-11-20" }, TODAY)).toBe("2026-11-21");
  });
});
