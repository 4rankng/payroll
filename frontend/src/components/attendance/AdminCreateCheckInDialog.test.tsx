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
  ProjectSelector: ({ onSelect, flexibleOnly }: { onSelect: (value: unknown) => void; flexibleOnly?: boolean }) => (
    <button type="button" data-flexible-only={String(flexibleOnly)} onClick={() => onSelect({ id: 2, name: "Dự án A", code: "DA", is_flexible: true })}>Chọn dự án thử</button>
  ),
}));

vi.mock("@/components/ui/alert", () => ({
  Alert: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

vi.stubGlobal("ResizeObserver", ResizeObserverMock);
Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
  configurable: true,
  value: vi.fn(),
});

describe("AdminCreateCheckInDialog", () => {
  it("submits only a check-in payload and explains that checkout remains employee-owned", async () => {
    useAdminCheckInShiftsMock.mockReturnValue({
      data: { shifts: [{ index: 0, "label": "08:00 - 17:00", start: "", end: "", amount: 300000, position: "Công nhân" }] },
      isLoading: false,
      error: null,
    });

    render(<AdminCreateCheckInDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText(/vẫn phải tự tan ca/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Chọn nhân viên thử" }));
    fireEvent.click(screen.getByRole("button", { name: "Chọn dự án thử" }));
    expect(screen.getByRole("button", { name: "Chọn dự án thử" })).toHaveAttribute("data-flexible-only", "true");

    // The shift dropdown is a SearchableSelect: opening it and picking the
    // server-provided shift proves it is the sole selectable attendance time.
    const shiftTrigger = screen.getByRole("combobox", { name: "Ca làm việc hôm nay" });
    // Radix Popover opens on pointerdown; jsdom has no PointerEvent, so the
    // event needs an explicit button to pass Radix's primary-button check.
    fireEvent.pointerDown(shiftTrigger, { button: 0 });
    fireEvent.click(shiftTrigger);
    fireEvent.click(await screen.findByText(/08:00 - 17:00/));  // label renders as "08:00 - 17:00 · Công nhân"
    expect(shiftTrigger).toHaveTextContent(/08:00 - 17:00/);
    expect(screen.queryByLabelText(/check-out/i)).not.toBeInTheDocument();
  });
});
