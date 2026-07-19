import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { format } from "date-fns";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AttendanceReference, EmployeeCheckInCard } from "./EmployeeCheckInCard";
import type { CheckInTarget } from "@/types/api/auth.types";
import type { LocationSample } from "@/utils/geolocation";

const locationMock = vi.hoisted(() => vi.fn());
const attendanceQueryMock = vi.hoisted(() => ({
  data: undefined as unknown,
  isLoading: false,
  isError: false,
  isFetching: false,
  refetch: vi.fn(),
}));

vi.mock("@/hooks/useContinuousLocation", () => ({
  isAbortedSubmitError: () => false,
  useContinuousLocation: locationMock,
}));

vi.mock("@/hooks/api/useAttendance", () => {
  const mutation = () => ({ isPending: false, mutate: vi.fn(), mutateAsync: vi.fn() });
  return {
    ATTENDANCE_QUERY_KEYS: { today: () => ["attendance", "today"] },
    useTodayAttendance: () => attendanceQueryMock,
    useCheckIn: mutation,
    useCheckOut: mutation,
    useCancelCurrentAttendance: mutation,
    useLogAttendanceDeviceAttempt: mutation,
  };
});

vi.mock("./EmployeeLocationMap", () => ({
  EmployeeLocationMap: ({ target, sample }: { target: CheckInTarget; sample?: LocationSample | null }) => (
    <div data-testid="employee-location-map">
      {target.project_name} · {sample ? Math.round(sample.accuracy) : "GPS"}
    </div>
  ),
}));

beforeEach(() => {
  attendanceQueryMock.data = undefined;
  attendanceQueryMock.isLoading = false;
  attendanceQueryMock.isError = false;
  attendanceQueryMock.isFetching = false;
  attendanceQueryMock.refetch.mockReset();
});

const localTime = (iso: string) => format(new Date(iso), "HH:mm");

describe("AttendanceReference", () => {
  it("shows the selected shift timing and configured checkpoints in the compact reference", () => {
    render(
      <AttendanceReference
        shiftStart="2026-07-12T08:00:00+07:00"
        shiftEnd="2026-07-12T17:00:00+07:00"
        checkInWindowStart="2026-07-12T07:00:00+07:00"
        checkInWindowEnd="2026-07-12T09:00:00+07:00"
        checkOutWindowStart="2026-07-12T16:00:00+07:00"
        checkOutWindowEnd="2026-07-12T21:00:00+07:00"
        checkInTarget={{
          project_id: 1,
          project_name: "Ting Ting Soft HQ",
          radius_meters: 150,
          gates: [
            { name: "Cổng chính", lat: 10.7769, lng: 106.7009 },
            { name: "Cổng kho", lat: 10.777, lng: 106.701 },
          ],
        }}
      />
    );

    expect(screen.getByRole("heading", { name: "Ca làm việc" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Khung giờ ca làm" })).toBeInTheDocument();
    expect(screen.getByText("Từ")).toBeInTheDocument();
    expect(screen.getByText("Đến")).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T08:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T17:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T07:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T09:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T16:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-12T21:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Các điểm chấm công" })).toHaveTextContent("Cổng chính");
    expect(screen.getByRole("list", { name: "Các điểm chấm công" })).toHaveTextContent("Cổng kho");
  });

  it("shows schedule values as device-local HH:mm instead of raw API timestamps", () => {
    const shiftStart = "2026-07-12T08:00:00+07:00";
    const shiftEnd = "2026-07-12T17:00:00+07:00";

    render(
      <AttendanceReference
        shiftStart={shiftStart}
        shiftEnd={shiftEnd}
        checkInWindowStart="2026-07-12T07:00:00+07:00"
        checkInWindowEnd="2026-07-12T09:00:00+07:00"
        checkOutWindowStart="2026-07-12T16:00:00+07:00"
        checkOutWindowEnd="2026-07-12T21:00:00+07:00"
      />
    );

    expect(screen.getByText(localTime(shiftStart))).toBeInTheDocument();
    expect(screen.getByText(localTime(shiftEnd))).toBeInTheDocument();
    expect(screen.queryByText(shiftStart)).not.toBeInTheDocument();
    expect(screen.queryByText(shiftEnd)).not.toBeInTheDocument();
  });

  it("shows overnight shift times as device-local HH:mm", () => {
    render(
      <AttendanceReference
        scheduleWindows={[
          {
            shift_start: "2026-07-12T22:00:00+07:00",
            shift_end: "2026-07-13T06:00:00+07:00",
            check_in_window_start: "2026-07-12T21:00:00+07:00",
            check_in_window_end: "2026-07-12T23:00:00+07:00",
            check_out_window_start: "2026-07-13T05:00:00+07:00",
            check_out_window_end: "2026-07-13T10:00:00+07:00",
          },
        ]}
      />
    );

    expect(screen.getByText(localTime("2026-07-12T22:00:00+07:00"))).toBeInTheDocument();
    expect(screen.getByText(localTime("2026-07-13T06:00:00+07:00"))).toBeInTheDocument();
    expect(screen.queryByText(/hôm sau/i)).not.toBeInTheDocument();
  });

  it("keeps the checkpoint reference compact and expands the other locations on demand", () => {
    render(
      <AttendanceReference
        checkInTarget={{
          project_id: 1,
          project_name: "Ting Ting Soft HQ",
          radius_meters: 150,
          gates: [
            { name: "Cổng chính", lat: 10.7769, lng: 106.7009 },
            { name: "Cổng kho", lat: 10.777, lng: 106.701 },
            { name: "Cổng phụ", lat: 10.7771, lng: 106.7011 },
            { name: "Cổng bãi xe", lat: 10.7772, lng: 106.7012 },
          ],
        }}
      />
    );

    const checkpoints = screen.getByRole("list", { name: "Các điểm chấm công" });
    expect(checkpoints).toHaveTextContent("Cổng chính");
    expect(checkpoints).toHaveTextContent("Cổng kho");
    expect(checkpoints).toHaveTextContent("+2");
    expect(checkpoints).not.toHaveTextContent("Cổng phụ");

    const moreLocations = screen.getByRole("button", { name: "Hiển thị 2 điểm chấm công khác" });
    expect(moreLocations).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(moreLocations);

    expect(moreLocations).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("list", { name: "Các điểm chấm công khác" })).toHaveTextContent("Cổng phụ");
    expect(screen.getByRole("list", { name: "Các điểm chấm công khác" })).toHaveTextContent("Cổng bãi xe");
  });

  it("selects the active shift and lets employees review another configured shift", () => {
    const morning = {
      shift_start: "2026-07-12T06:00:00+07:00",
      shift_end: "2026-07-12T14:00:00+07:00",
      check_in_window_start: "2026-07-12T05:00:00+07:00",
      check_in_window_end: "2026-07-12T07:00:00+07:00",
      check_out_window_start: "2026-07-12T13:00:00+07:00",
      check_out_window_end: "2026-07-12T18:00:00+07:00",
    };
    const evening = {
      shift_start: "2026-07-12T14:00:00+07:00",
      shift_end: "2026-07-12T22:00:00+07:00",
      check_in_window_start: "2026-07-12T13:00:00+07:00",
      check_in_window_end: "2026-07-12T15:00:00+07:00",
      check_out_window_start: "2026-07-12T21:00:00+07:00",
      check_out_window_end: "2026-07-13T02:00:00+07:00",
    };

    render(<AttendanceReference scheduleWindows={[morning, evening]} activeScheduleWindow={evening} />);

    const firstShiftTab = screen.getByRole("tab", { name: "Ca ngày" });
    const secondShiftTab = screen.getByRole("tab", { name: "Ca đêm" });

    expect(secondShiftTab).toHaveAttribute("aria-selected", "true");
    expect(secondShiftTab).toHaveAttribute("aria-controls");
    expect(screen.getByRole("tabpanel")).toHaveAttribute("aria-labelledby", secondShiftTab.id);
    expect(screen.getByText(localTime(evening.shift_start))).toBeInTheDocument();
    expect(screen.getByText(localTime(evening.shift_end))).toBeInTheDocument();
    expect(screen.getByText(localTime(evening.check_out_window_end))).toBeInTheDocument();

    fireEvent.click(firstShiftTab);

    expect(firstShiftTab).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText(localTime(morning.shift_start))).toBeInTheDocument();
    expect(screen.getByText(localTime(morning.check_in_window_start))).toBeInTheDocument();
    expect(screen.getByText(localTime(morning.check_out_window_end))).toBeInTheDocument();
  });

  it("uses admin-configured shift_name for tab labels when provided", () => {
    const day = {
      shift_start: "2026-07-12T09:00:00+07:00",
      shift_end: "2026-07-12T18:00:00+07:00",
      check_in_window_start: "2026-07-12T08:00:00+07:00",
      check_in_window_end: "2026-07-12T10:00:00+07:00",
      check_out_window_start: "2026-07-12T17:00:00+07:00",
      check_out_window_end: "2026-07-12T22:00:00+07:00",
      shift_name: "Ca làm",
    };
    const night = {
      shift_start: "2026-07-12T21:00:00+07:00",
      shift_end: "2026-07-13T05:00:00+07:00",
      check_in_window_start: "2026-07-12T20:00:00+07:00",
      check_in_window_end: "2026-07-12T22:00:00+07:00",
      check_out_window_start: "2026-07-13T04:00:00+07:00",
      check_out_window_end: "2026-07-13T09:00:00+07:00",
      shift_name: "Ca đêm",
    };

    render(<AttendanceReference scheduleWindows={[day, night]} activeScheduleWindow={day} />);

    // Admin names replace the legacy "Ca ngày"/"Ca đêm" labels.
    expect(screen.getByRole("tab", { name: /Ca làm/ })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Ca đêm/ })).toBeInTheDocument();
    // Legacy fallback labels should NOT appear.
    expect(screen.queryByRole("tab", { name: "Ca ngày" })).not.toBeInTheDocument();
  });

  it("shows only shift names without an overnight badge", () => {
    const day = {
      shift_start: "2026-07-12T09:00:00+07:00",
      shift_end: "2026-07-12T18:00:00+07:00",
      check_in_window_start: "2026-07-12T08:00:00+07:00",
      check_in_window_end: "2026-07-12T10:00:00+07:00",
      check_out_window_start: "2026-07-12T17:00:00+07:00",
      check_out_window_end: "2026-07-12T22:00:00+07:00",
    };
    const night = {
      shift_start: "2026-07-12T21:00:00+07:00",
      shift_end: "2026-07-13T05:00:00+07:00",
      check_in_window_start: "2026-07-12T20:00:00+07:00",
      check_in_window_end: "2026-07-12T22:00:00+07:00",
      check_out_window_start: "2026-07-13T04:00:00+07:00",
      check_out_window_end: "2026-07-13T09:00:00+07:00",
    };

    render(<AttendanceReference scheduleWindows={[day, night]} activeScheduleWindow={night} />);

    const nightTab = screen.getByRole("tab", { name: "Ca đêm" });
    expect(nightTab).toHaveTextContent("Ca đêm");
    const dayTab = screen.getByRole("tab", { name: "Ca ngày" });
    expect(dayTab).toHaveTextContent("Ca ngày");
    expect(screen.queryByText("Qua đêm")).not.toBeInTheDocument();
  });
});

describe("EmployeeCheckInCard geofence guidance", () => {
  it("blocks attendance mutations and offers retry when today's state is unknown", () => {
    attendanceQueryMock.isError = true;
    locationMock.mockReturnValue({
      sample: null,
      progress: null,
      isSubmitReady: false,
      isWatching: false,
      fatalError: null,
      awaitSubmitReady: vi.fn(),
      awaitAccurateSample: vi.fn(),
      retry: vi.fn(),
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <EmployeeCheckInCard onAdvanceRequest={vi.fn()} />
      </QueryClientProvider>
    );

    expect(screen.getByRole("heading", { name: "Chưa tải được trạng thái chấm công" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Chưa tải được chấm công" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Tải lại chấm công" }));
    expect(attendanceQueryMock.refetch).toHaveBeenCalledOnce();
  });

  it("keeps a retained attendance state usable when a background refetch fails", () => {
    attendanceQueryMock.data = {
      data: {
        id: 9,
        project_id: 2,
        employee_id: 77,
        date: "2026-07-19",
        check_in_time: "2026-07-19T08:00:00+07:00",
        check_in_gate: "Cổng chính",
        status: "checked_in",
      },
    };
    attendanceQueryMock.isError = true;
    locationMock.mockReturnValue({
      sample: null,
      progress: null,
      isSubmitReady: false,
      isWatching: false,
      fatalError: null,
      awaitSubmitReady: vi.fn(),
      awaitAccurateSample: vi.fn(),
      retry: vi.fn(),
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <EmployeeCheckInCard onAdvanceRequest={vi.fn()} />
      </QueryClientProvider>
    );

    expect(screen.queryByRole("heading", { name: "Chưa tải được trạng thái chấm công" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Tan ca" }).some((button) => !button.hasAttribute("disabled"))).toBe(true);
  });

  it("shows inward guidance and opens the map for an inside-but-uncertain fix", async () => {
    const gate = { name: "Cổng D", lat: 20.8679818, lng: 106.5711738 };
    const sample = {
      lat: gate.lat + 124 / 111_195,
      lng: gate.lng,
      accuracy: 48,
      timestamp: Date.now(),
    };
    locationMock.mockReturnValue({
      sample,
      progress: {
        sampleCount: 2,
        elapsedMs: 3_000,
        bestAccuracy: 48,
        requiredAccuracyMeters: 150,
        status: "acceptable",
      },
      isSubmitReady: false,
      isWatching: false,
      fatalError: null,
      awaitSubmitReady: vi.fn(),
      awaitAccurateSample: vi.fn(),
      retry: vi.fn(),
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <EmployeeCheckInCard
          checkInTarget={{
            project_id: 58,
            project_name: "LGD",
            radius_meters: 150,
            gates: [gate],
          }}
          onAdvanceRequest={vi.fn()}
        />
      </QueryClientProvider>
    );

    expect(screen.getByText("Tiến gần tâm khu vực")).toBeInTheDocument();
    expect(
      screen.getByText("Hãy tiến gần hơn tới tâm khu vực chấm công tại Cổng D rồi thử lại.")
    ).toBeInTheDocument();
    expect(screen.queryByText("Sẵn sàng vào làm")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ẩn bản đồ" })).toHaveAttribute("aria-expanded", "true");
    expect(await screen.findByTestId("employee-location-map")).toBeInTheDocument();
  });

  it("respects a manual close until the boundary guidance episode ends and begins again", async () => {
    const gate = { name: "Cổng D", lat: 20.8679818, lng: 106.5711738 };
    let sample: LocationSample = {
      lat: gate.lat + 124 / 111_195,
      lng: gate.lng,
      accuracy: 48,
      timestamp: Date.now(),
    };
    locationMock.mockImplementation(() => ({
      sample,
      progress: {
        sampleCount: 2,
        elapsedMs: 3_000,
        bestAccuracy: sample.accuracy,
        requiredAccuracyMeters: 150,
        status: "acceptable",
      },
      isSubmitReady: false,
      isWatching: false,
      fatalError: null,
      awaitSubmitReady: vi.fn(),
      awaitAccurateSample: vi.fn(),
      retry: vi.fn(),
    }));

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const renderCard = () => (
      <QueryClientProvider client={queryClient}>
        <EmployeeCheckInCard
          checkInTarget={{
            project_id: 58,
            project_name: "LGD",
            radius_meters: 150,
            gates: [gate],
          }}
          onAdvanceRequest={vi.fn()}
        />
      </QueryClientProvider>
    );
    const { rerender } = render(renderCard());

    const disclosure = screen.getByRole("button", { name: "Ẩn bản đồ" });
    expect(await screen.findByTestId("employee-location-map")).toBeInTheDocument();
    fireEvent.click(disclosure);
    expect(screen.getByRole("button", { name: "Xem bản đồ" })).toHaveAttribute("aria-expanded", "false");

    sample = { ...sample, lat: gate.lat + 122 / 111_195, timestamp: sample.timestamp + 1_000 };
    rerender(renderCard());
    expect(screen.getByRole("button", { name: "Xem bản đồ" })).toHaveAttribute("aria-expanded", "false");

    sample = { ...sample, accuracy: 800, timestamp: sample.timestamp + 1_000 };
    rerender(renderCard());
    expect(screen.getByRole("button", { name: "Xem bản đồ" })).toHaveAttribute("aria-expanded", "false");

    sample = { ...sample, accuracy: 48, timestamp: sample.timestamp + 1_000 };
    rerender(renderCard());
    expect(screen.getByRole("button", { name: "Ẩn bản đồ" })).toHaveAttribute("aria-expanded", "true");
  });

  it("keeps genuinely poor GPS guidance collapsed instead of presenting a direction", () => {
    const gate = { name: "Cổng D", lat: 20.8679818, lng: 106.5711738 };
    locationMock.mockReturnValue({
      sample: { lat: gate.lat, lng: gate.lng, accuracy: 800, timestamp: Date.now() },
      progress: {
        sampleCount: 2,
        elapsedMs: 3_000,
        bestAccuracy: 800,
        requiredAccuracyMeters: 150,
        status: "poor",
      },
      isSubmitReady: false,
      isWatching: true,
      fatalError: null,
      awaitSubmitReady: vi.fn(),
      awaitAccurateSample: vi.fn(),
      retry: vi.fn(),
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <EmployeeCheckInCard
          checkInTarget={{ project_id: 58, project_name: "LGD", radius_meters: 150, gates: [gate] }}
          onAdvanceRequest={vi.fn()}
        />
      </QueryClientProvider>
    );

    expect(screen.getByRole("button", { name: "Xem bản đồ" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Tiến gần tâm khu vực")).not.toBeInTheDocument();
  });
});
