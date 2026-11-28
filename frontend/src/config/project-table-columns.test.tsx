import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { createProjectColumns } from "./project-table-columns";
import { createProjectMobileConfig } from "./project-table-mobile";
import type { Project } from "@/types/api/project.types";
import type { ColumnDef } from "@tanstack/react-table";

const project = {
  id: 1,
  name: "Dự án A",
  code: "DA001",
  status: "active",
  employee_count: 12,
  weekly_salary_employee_count: 0,
  monthly_salary_employee_count: 0,
} as unknown as Project;

// Drive the cell/render functions directly — the fallback lives there, and the
// table wrappers are covered by the workspace-parity tests.
function desktopCell(current: Project) {
  const columns = createProjectColumns();
  const column = columns.find((c) => c.id === "employee_summary") as ColumnDef<Project> & {
    cell: (ctx: { row: { original: Project } }) => React.ReactNode;
  };
  return render(column.cell({ row: { original: current } }));
}

function mobileCell(current: Project) {
  const { mobileFields } = createProjectMobileConfig();
  const field = mobileFields.find((f) => f.key === "employee_summary")!;
  return render(field.render(current));
}

describe("project table NHÂN VIÊN column fallback (PAYR-114)", () => {
  it("desktop binds employee_count instead of a blank dash when the schedule split is 0", () => {
    desktopCell(project);
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("mobile binds employee_count instead of a blank dash when the schedule split is 0", () => {
    mobileCell(project);
    expect(screen.getByText("12")).toBeInTheDocument();
  });

  it("renders 0 when the project has no employees at all", () => {
    desktopCell({ ...project, employee_count: 0 });
    expect(screen.getByText("0")).toBeInTheDocument();
  });
});
