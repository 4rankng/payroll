import { fireEvent, render, screen } from "@testing-library/react";
import { format } from "date-fns";
import { describe, expect, it } from "vitest";
import { AttendanceReference } from "./EmployeeCheckInCard";

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
});
