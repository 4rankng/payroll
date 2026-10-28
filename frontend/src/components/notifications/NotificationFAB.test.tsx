import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { NotificationFAB } from "./NotificationFAB";

const mocks = vi.hoisted(() => ({
  isMobile: vi.fn(() => false),
  unreadCount: 11,
}));

vi.mock("@/hooks/useBreakpoint", () => ({
  useIsMobile: mocks.isMobile,
}));
vi.mock("@/hooks/api/useNotifications", () => ({
  useUnreadNotifications: () => ({ data: { count: mocks.unreadCount }, isLoading: false }),
}));
vi.mock("@/hooks/usePushNotifications", () => ({
  usePushNotifications: () => ({
    isSubscribed: false,
    isSupported: true,
    permissionStatus: "default",
    requestPermission: vi.fn(),
  }),
}));
vi.mock("@/hooks/useBellAnimation", () => ({
  useBellAnimation: () => "bell.png",
}));
vi.mock("@/lib/auth", () => ({
  authManager: { getUserRole: () => "admin" },
}));
vi.mock("@/components/notifications/NotificationSheet", () => ({
  NotificationSheet: () => null,
}));

describe("NotificationFAB không đè nội dung (PAYR-28)", () => {
  it("badge nằm gọn trong hit area FAB (không overhang -top-1 -right-1)", () => {
    render(<NotificationFAB />);
    const fab = screen.getByRole("button", { name: /Thông báo/ });
    const badge = screen.getByText("11");
    // Badge là con của button → click trên badge vẫn nằm trong hit area FAB.
    expect(fab.contains(badge)).toBe(true);
    expect(badge.className).toContain("top-1");
    expect(badge.className).toContain("right-1");
    expect(badge.className).not.toContain("-top-1");
    expect(badge.className).not.toContain("-right-1");
  });

  it("mobile (<768px) không render FAB — mobile nav đã có notifications", () => {
    mocks.isMobile.mockReturnValue(true);
    const { container } = render(<NotificationFAB />);
    expect(container.querySelector("button")).toBeNull();
    mocks.isMobile.mockReturnValue(false);
  });

  it("content frame có FAB clearance rail ≥768px (admin + partner)", () => {
    const adminCss = readFileSync(resolve(process.cwd(), "src/styles/admin-daisy.css"), "utf8");
    expect(adminCss).toMatch(
      /@media \(min-width: 768px\)[\s\S]{0,200}admin-page-frame\s*\{\s*padding-right: 4\.5rem;/,
    );
    const baseCss = readFileSync(resolve(process.cwd(), "src/styles/base.css"), "utf8");
    expect(baseCss).toMatch(
      /@media \(min-width: 768px\)[\s\S]{0,200}mobile-main-content\s*\{\s*padding-right: 4\.5rem;/,
    );
  });
});
