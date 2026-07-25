import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Project } from "@/types/api/project.types";
import { ProjectMobileList } from "./ProjectMobileList";

const project = {
  id: 42,
  name: "Dự án thử nghiệm",
  code: "DA-42",
  status: "active",
  employee_count: 5,
} as Project;

describe("ProjectMobileList", () => {
  it("keeps the optional project-scoped timesheet action separate from details", () => {
    const onRowClick = vi.fn();
    const onTimesheet = vi.fn();

    render(
      <ProjectMobileList
        projects={[project]}
        onRowClick={onRowClick}
        onTimesheet={onTimesheet}
      />,
    );

    const timesheetButton = screen.getByRole("button", { name: "Bảng công" });
    expect(timesheetButton).toHaveClass("min-h-11");

    fireEvent.click(timesheetButton);

    expect(onTimesheet).toHaveBeenCalledWith(project);
    expect(onRowClick).not.toHaveBeenCalled();

    fireEvent.keyDown(timesheetButton, { key: " " });
    expect(onRowClick).not.toHaveBeenCalled();
  });
});
