import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AdminCreateCheckInDialog } from "./AdminCreateCheckInDialog";

const { useAdminCheckInShiftsMock } = vi.hoisted(() => ({
  useAdminCheckInShiftsMock: vi.fn(),
}));

vi.mock("@/hooks/api/useAdminAttendance", () => ({
  useAdminCheckInShifts: (...args: unknown[]) => useAdminCheckInShiftsMock(...args),
  useAdminCreateCheckIn: () => ({ isPending: false, mutate: vi.fn() }),
}));

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: React.ReactNode }) => <h2>{children}</h2>,
  DialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/ui/employee-selector", () => ({
  EmployeeSelector: ({ onSelect }: { onSelect: (value: unknown) => void }) => (
    <button type="button" onClick={() => onSelect({ id: 3, fullname: "Trần Thị Bình", cccd: "001" })}>Chọn nhân viên thử</button>
  ),
}));

vi.mock("@/components/ui/project-selector", () => ({
  ProjectSelector: ({ onSelect }: { onSelect: (value: unknown) => void }) => (
    <button type="button" onClick={() => onSelect({ id: 2, name: "Dự án A", code: "DA" })}>Chọn dự án thử</button>
  ),
}));

vi.mock("@/components/ui/select", () => ({
  Select: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SelectTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SelectValue: () => null,
  SelectContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  SelectItem: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("@/components/ui/alert", () => ({
  Alert: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

describe("AdminCreateCheckInDialog", () => {
  it("submits only a check-in payload and explains that checkout remains employee-owned", () => {
    useAdminCheckInShiftsMock.mockReturnValue({
      data: { shifts: [{ index: 0, label: "08:00 - 17:00", start: "", end: "", amount: 300000, position: "Công nhân" }] },
      isLoading: false,
      error: null,
    });

    render(<AdminCreateCheckInDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText(/vẫn phải tự tan ca/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Chọn nhân viên thử" }));
    fireEvent.click(screen.getByRole("button", { name: "Chọn dự án thử" }));

    // The rendered shift button proves the server-provided shift is the sole
    // selectable attendance time; no checkout input is present in this form.
    expect(screen.getByRole("button", { name: /08:00 - 17:00/ })).toBeInTheDocument();
    expect(screen.queryByLabelText(/check-out/i)).not.toBeInTheDocument();
  });
});
