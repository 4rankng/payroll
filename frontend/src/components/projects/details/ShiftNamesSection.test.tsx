import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ShiftNamesSection } from "./ShiftNamesSection";
import type { Project } from "@/types/api/project.types";

const { mockUseProjectPayRates, mockUpdateProject } = vi.hoisted(() => ({
  mockUseProjectPayRates: vi.fn(),
  mockUpdateProject: vi.fn(),
}));

vi.mock("@/hooks/api/usePayRates", () => ({
  useProjectPayRates: mockUseProjectPayRates,
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useUpdateProject: () => ({
    mutate: mockUpdateProject,
    isPending: false,
  }),
}));

const project = {
  id: 58,
  name: "Dự án thử nghiệm",
  code: "DA58",
  client_name: "Khách hàng",
  start_date: "2026-01-01",
  end_date: null,
  employee_count: 0,
  weekly_salary_employee_count: 0,
  monthly_salary_employee_count: 0,
  status: "active",
  is_flexible: true,
  shift_names: [{ range: "08:00-17:00", name: "Ca ngày" }],
  created_by: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
} satisfies Project;

describe("ShiftNamesSection", () => {
  beforeEach(() => {
    mockUpdateProject.mockReset();
    mockUseProjectPayRates.mockReturnValue({
      data: {
        data: [
          {
            id: 91,
            project_id: project.id,
            fromDate: "2020-01-01",
            toDate: null,
            rates: {
              "Công nhân": {
                "ngày thường": {
                  "08:00-17:00": 300000,
                  "20:00-08:00": 350000,
                },
              },
            },
          },
        ],
      },
    });
  });

  it("opens the authoritative payrate editor from the shift table", () => {
    const onEditPayrate = vi.fn();

    render(
      <ShiftNamesSection
        project={project}
        canEdit
        onEditPayrate={onEditPayrate}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Chỉnh giờ & lương" }));

    expect(onEditPayrate).toHaveBeenCalledWith(91);
    expect(screen.getByText("20:00-08:00")).toBeInTheDocument();
    expect(screen.getByText("Qua đêm")).toBeInTheDocument();
  });

  it("keeps the shift ranges read-only without edit permission", () => {
    render(
      <ShiftNamesSection
        project={project}
        canEdit={false}
        onEditPayrate={vi.fn()}
      />
    );

    expect(screen.queryByRole("button", { name: "Chỉnh giờ & lương" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Đặt tên ca" })).not.toBeInTheDocument();
    expect(screen.getByText("08:00-17:00")).toBeInTheDocument();
  });

  it("opens the nearest upcoming payrate when no configuration is active", () => {
    const onEditPayrate = vi.fn();
    mockUseProjectPayRates.mockReturnValue({
      data: {
        data: [
          {
            id: 10,
            fromDate: "2020-01-01",
            toDate: "2020-12-31",
            rates: { "Công nhân": { "ngày thường": { "06:00-14:00": 200000 } } },
          },
          {
            id: 30,
            fromDate: "2100-01-01",
            toDate: null,
            rates: { "Công nhân": { "ngày thường": { "07:00-15:00": 250000 } } },
          },
          {
            id: 20,
            fromDate: "2090-01-01",
            toDate: null,
            rates: { "Công nhân": { "ngày thường": { "08:00-16:00": 300000 } } },
          },
        ],
      },
    });

    render(
      <ShiftNamesSection
        project={project}
        canEdit
        onEditPayrate={onEditPayrate}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Chỉnh giờ & lương" }));

    expect(onEditPayrate).toHaveBeenCalledWith(20);
    expect(screen.getByText("08:00-16:00")).toBeInTheDocument();
  });

  it("keeps the edit action available before the first shift range is added", () => {
    const onEditPayrate = vi.fn();
    mockUseProjectPayRates.mockReturnValue({
      data: {
        data: [
          {
            id: 92,
            fromDate: "2020-01-01",
            toDate: null,
            rates: {},
          },
        ],
      },
    });

    render(
      <ShiftNamesSection
        project={project}
        canEdit
        onEditPayrate={onEditPayrate}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Chỉnh giờ & lương" }));

    expect(onEditPayrate).toHaveBeenCalledWith(92);
    expect(screen.getByText(/mức lương theo giờ/)).toBeInTheDocument();
  });
});
