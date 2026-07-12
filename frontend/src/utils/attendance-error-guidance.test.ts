import { describe, expect, it, vi } from "vitest";
import {
  getTimingErrorGuidance,
  hasLocationErrorGuidance,
  scrollToLocationGuidance,
} from "./attendance-error-guidance";

describe("attendance error guidance", () => {
  it("uses structured check-in timing guidance", () => {
    expect(
      getTimingErrorGuidance(
        {
          status: "error",
          message: "Giờ vào làm không hợp lệ. Bạn chỉ được vào làm từ 07:00 đến 09:00.",
          details: { guidance_type: "timing", action: "check_in", window_start: "07:00", window_end: "09:00" },
        },
        "check_in"
      )
    ).toMatchObject({ title: "Chưa đến giờ vào làm", windowStart: "07:00", windowEnd: "09:00" });
  });

  it("falls back to the existing timing message", () => {
    expect(
      getTimingErrorGuidance(
        { status: "error", message: "Giờ vào làm không hợp lệ. Bạn chỉ được vào làm từ 07:00 đến 09:00." },
        "check_in"
      )
    ).toMatchObject({ windowStart: "07:00", windowEnd: "09:00" });
  });

  it("identifies backend location guidance", () => {
    expect(hasLocationErrorGuidance({ status: "error", message: "Ngoài khu vực", details: { guidance_type: "location" } })).toBe(true);
  });

  it("moves the employee directly to the expanded map guidance", () => {
    const element = document.createElement("div");
    const scrollIntoView = vi.fn();
    Object.defineProperty(element, "scrollIntoView", { value: scrollIntoView });
    const requestAnimationFrame = vi
      .spyOn(window, "requestAnimationFrame")
      .mockImplementation((callback) => {
        callback(0);
        return 1;
      });

    scrollToLocationGuidance(element);

    expect(requestAnimationFrame).toHaveBeenCalledOnce();
    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
    requestAnimationFrame.mockRestore();
  });

  it("avoids animated scrolling when reduced motion is requested", () => {
    const element = document.createElement("div");
    const scrollIntoView = vi.fn();
    Object.defineProperty(element, "scrollIntoView", { value: scrollIntoView });
    const matchMedia = vi.spyOn(window, "matchMedia").mockReturnValue({ matches: true } as MediaQueryList);
    const requestAnimationFrame = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      callback(0);
      return 1;
    });

    scrollToLocationGuidance(element);

    expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "auto", block: "center" });
    matchMedia.mockRestore();
    requestAnimationFrame.mockRestore();
  });
});
