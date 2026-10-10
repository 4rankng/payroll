import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CheckInRosterReminderBanner } from "./CheckInRosterReminderBanner";
import type { Notification } from "@/types/api/notification.types";
import type { AssetsResponse } from "@/types/api/financial.types";

const notificationsMock = vi.hoisted(() => ({
  data: undefined as { notifications: Notification[]; count: number } | undefined,
}));
const assetsMock = vi.hoisted(() => ({
  data: undefined as AssetsResponse | undefined,
  isLoading: false,
  refetch: vi.fn(),
}));
const markAsReadMock = vi.hoisted(() => ({
  mutate: vi.fn(),
  isPending: false,
}));
const downloadAssetFileMock = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/api/useNotifications", () => ({
  useUnreadNotifications: () => notificationsMock,
  useMarkAsReadSilent: () => markAsReadMock,
}));
vi.mock("@/hooks/api/useAssets", () => ({
  useAssets: () => assetsMock,
}));
vi.mock("@/utils/file-download", () => ({
  downloadAssetFile: downloadAssetFileMock,
}));

function makeNotice(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 1,
    type: "checkin_roster",
    channel: "push",
    title: "Danh sách tự chấm công đã sẵn sàng",
    message:
      "Danh sách nhân viên tự chấm công tháng 09 (12 người) đã được tạo. Tải file tại Tổng quan và gửi cho đối tác.",
    read_at: null,
    created_at: "2026-10-09T02:15:00Z",
    ...overrides,
  };
}

function makeAssetsResponse(overrides: Partial<AssetsResponse> = {}): AssetsResponse {
  return {
    status: "success",
    message: "OK",
    data: [
      {
        id: 42,
        filename: "danh_sach_tu_cham_cong_2026-09.xlsx",
        upload_type: "checkin_roster",
        uploaded_by: 49,
        created_at: "2026-10-09T02:14:00Z",
      },
    ],
    pagination: { page: 1, pageSize: 1, totalPages: 1, totalRecords: 1 },
    ...overrides,
  };
}

describe("CheckInRosterReminderBanner", () => {
  beforeEach(() => {
    notificationsMock.data = { notifications: [], count: 0 };
    assetsMock.data = makeAssetsResponse();
    assetsMock.isLoading = false;
    assetsMock.refetch.mockClear();
    markAsReadMock.mutate.mockClear();
    markAsReadMock.isPending = false;
    downloadAssetFileMock.mockClear();
  });

  it("renders nothing when there is no unread roster notice", () => {
    notificationsMock.data = {
      notifications: [makeNotice({ type: "custom", id: 9 })],
      count: 1,
    };
    const { container } = render(<CheckInRosterReminderBanner />);
    expect(container.innerHTML).toBe("");
  });

  it("shows the newest roster notice copy with both actions", () => {
    notificationsMock.data = {
      notifications: [
        makeNotice({ id: 1, created_at: "2026-09-09T02:15:00Z", message: "tháng 08 cũ" }),
        makeNotice({ id: 2, created_at: "2026-10-09T02:15:00Z", message: "tháng 09 mới" }),
      ],
      count: 2,
    };
    render(<CheckInRosterReminderBanner />);

    expect(
      screen.getByText("Danh sách tự chấm công đã sẵn sàng")
    ).toBeInTheDocument();
    expect(screen.getByText("tháng 09 mới")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tải file" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Đã xong" })).toBeEnabled();
  });

  it("downloads the prepared workbook when Tải file is clicked", () => {
    notificationsMock.data = {
      notifications: [makeNotice()],
      count: 1,
    };
    render(<CheckInRosterReminderBanner />);

    fireEvent.click(screen.getByRole("button", { name: "Tải file" }));
    expect(downloadAssetFileMock).toHaveBeenCalledWith(
      42,
      "danh_sach_tu_cham_cong_2026-09.xlsx"
    );
  });

  it("shows a loading label until the asset list has loaded", () => {
    notificationsMock.data = { notifications: [makeNotice()], count: 1 };
    assetsMock.data = undefined;
    assetsMock.isLoading = true;
    render(<CheckInRosterReminderBanner />);

    expect(
      screen.getByRole("button", { name: "Đang tải file…" })
    ).toBeDisabled();
  });

  it("retries the asset lookup when no workbook is found yet", async () => {
    vi.useFakeTimers();
    try {
      notificationsMock.data = { notifications: [makeNotice()], count: 1 };
      assetsMock.data = undefined;
      assetsMock.isLoading = false;
      render(<CheckInRosterReminderBanner />);

      expect(assetsMock.refetch).not.toHaveBeenCalled();
      await vi.advanceTimersByTimeAsync(1500);
      expect(assetsMock.refetch).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("does not retry once the workbook is available", () => {
    notificationsMock.data = { notifications: [makeNotice()], count: 1 };
    render(<CheckInRosterReminderBanner />);

    expect(assetsMock.refetch).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Tải file" })).toBeEnabled();
  });

  it("marks every roster notice read on Đã xong, leaving other types alone", () => {
    notificationsMock.data = {
      notifications: [
        makeNotice({ id: 1, created_at: "2026-09-09T02:15:00Z" }),
        makeNotice({ id: 2, created_at: "2026-10-09T02:15:00Z" }),
        makeNotice({ id: 3, type: "custom" }),
      ],
      count: 3,
    };
    render(<CheckInRosterReminderBanner />);

    fireEvent.click(screen.getByRole("button", { name: "Đã xong" }));
    const markedIds = markAsReadMock.mutate.mock.calls.map((c) => c[0]);
    expect(markedIds).toEqual([2, 1]);
    expect(markedIds).not.toContain(3);
  });

  it("marks read from the dismiss button too", () => {
    notificationsMock.data = { notifications: [makeNotice({ id: 7 })], count: 1 };
    render(<CheckInRosterReminderBanner />);

    fireEvent.click(screen.getByRole("button", { name: "Đánh dấu đã xong" }));
    expect(markAsReadMock.mutate).toHaveBeenCalledWith(7);
  });
});
