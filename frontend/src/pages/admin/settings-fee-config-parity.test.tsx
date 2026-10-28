import { cleanup, render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import SettingsPage from "./SettingsPage";
import SettingsPageMobile from "@/pages/mobile/admin/SettingsPage";

const mocks = vi.hoisted(() => {
  const order: string[] = [];
  return {
    order,
    record: (name: string) => () => {
      order.push(name);
      return null;
    },
  };
});

vi.mock("@/hooks/settings/useSettingsForm", () => ({
  useSettingsForm: () => ({ isLoading: false, form: {} }),
}));
vi.mock("@/components/settings/SettingsTabList", async (importOriginal) => ({
  ...((await importOriginal()) as object),
  SettingsTabList: () => null,
}));
vi.mock("@/components/shared/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/shared/AdminPageFrame", () => ({
  AdminPageCanvas: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminPageHeaderCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection", () => ({
  FeeScheduleSection: mocks.record("FeeScheduleSection"),
}));
vi.mock("@/components/admin/WeeklyPaymentFeeSchedule/WeeklyPaymentFeeScheduleSection", () => ({
  WeeklyPaymentFeeScheduleSection: mocks.record("WeeklyPaymentFeeScheduleSection"),
}));
vi.mock("@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection", () => ({
  DisbursementFeeScheduleSection: mocks.record("DisbursementFeeScheduleSection"),
}));
vi.mock("@/components/settings/SettingsGeneralPanel", () => ({
  SettingsGeneralPanel: mocks.record("SettingsGeneralPanel"),
}));
vi.mock("@/components/settings/SettingsEmployeePanel", () => ({
  SettingsEmployeePanel: mocks.record("SettingsEmployeePanel"),
}));
vi.mock("@/components/email/AdminEmailComposer", () => ({
  AdminEmailComposer: mocks.record("AdminEmailComposer"),
}));
vi.mock("@/components/settings/SendNotificationComposer", () => ({
  SendNotificationComposer: mocks.record("SendNotificationComposer"),
}));
vi.mock("@/components/settings/ZaloConnectionSection", () => ({
  ZaloConnectionSection: mocks.record("ZaloConnectionSection"),
}));
vi.mock("@/components/settings/AdBannerSection", () => ({
  AdBannerSection: mocks.record("AdBannerSection"),
}));
vi.mock("@/components/settings/ApiKeysSection", () => ({
  ApiKeysSection: mocks.record("ApiKeysSection"),
}));
vi.mock("@/components/settings/McpApiKeysSection", () => ({
  McpApiKeysSection: mocks.record("McpApiKeysSection"),
}));
vi.mock("@/components/settings/OpsApiKeysSection", () => ({
  OpsApiKeysSection: mocks.record("OpsApiKeysSection"),
}));

function renderSectionOrder(node: React.ReactNode): string[] {
  mocks.order.length = 0;
  render(
    <MemoryRouter initialEntries={["/?tab=fee-config"]}>{node}</MemoryRouter>,
  );
  return [...mocks.order];
}

describe("Settings fee-config parity desktop ↔ mobile (PAYR-21)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("section list mobile == desktop, gồm đủ WeeklyPaymentFeeScheduleSection", () => {
    const desktop = renderSectionOrder(<SettingsPage />);
    cleanup();
    const mobile = renderSectionOrder(<SettingsPageMobile />);

    expect(desktop).toEqual([
      "FeeScheduleSection",
      "WeeklyPaymentFeeScheduleSection",
      "DisbursementFeeScheduleSection",
    ]);
    expect(mobile).toEqual(desktop);
  });
});
