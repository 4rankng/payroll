import { describe, expect, it } from "vitest";
import {
  getFailedAttemptOverrideViewState,
  isGpsDeviceFailure,
} from "./failedAttemptOverride";

describe("failed attempt override labels", () => {
  it("does not label an unresolved GPS failure as recorded", () => {
    const state = getFailedAttemptOverrideViewState("gps_denied", null);

    expect(state.canOverride).toBe(true);
    expect(state.isResolved).toBe(false);
    expect(state.actionLabel).toBe("Cần duyệt GPS");
    expect(state.actionLabel).not.toBe("Ghi nhận chấm công");
  });

  it("marks a GPS failure as recorded only after resolution", () => {
    const state = getFailedAttemptOverrideViewState("gps_denied", "2026-07-02T10:01:00Z");

    expect(state.canOverride).toBe(true);
    expect(state.isResolved).toBe(true);
    expect(state.resolvedLabel).toBe("Đã ghi nhận");
  });

  it("only allows device GPS failures to be overridden", () => {
    expect(isGpsDeviceFailure("gps_timeout")).toBe(true);
    expect(isGpsDeviceFailure("geofence_outside")).toBe(false);
  });
});
