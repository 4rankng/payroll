import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AdvanceRequestToggle } from "./AdvanceRequestToggle";

const mutateAsyncMock = vi.hoisted(() => vi.fn(() => Promise.resolve()));

vi.mock("@/hooks/api/useAdvancePayments", () => ({
  useToggleAdvanceRequestEnabled: () => ({ isPending: false, mutateAsync: mutateAsyncMock }),
}));

const renderToggle = (enabled: boolean, onParentClick?: () => void) =>
  render(
    <div onClick={onParentClick}>
      <AdvanceRequestToggle projectId={7} employeeId={42} enabled={enabled} />
    </div>,
  );

beforeEach(() => {
  mutateAsyncMock.mockReset();
  mutateAsyncMock.mockResolvedValue(undefined);
});

describe("AdvanceRequestToggle", () => {
  it("pausing requires confirmation before firing the mutation", () => {
    renderToggle(true);

    fireEvent.click(screen.getByRole("switch"));

    expect(mutateAsyncMock).not.toHaveBeenCalled();
    expect(screen.getByText("Tạm ngừng ứng lương?")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Tạm ngừng" }));
    expect(mutateAsyncMock).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 42,
      enabled: false,
    });
  });

  it("cancelling the dialog leaves the switch untouched", () => {
    renderToggle(true);

    fireEvent.click(screen.getByRole("switch"));
    fireEvent.click(screen.getByRole("button", { name: "Giữ lại" }));

    expect(mutateAsyncMock).not.toHaveBeenCalled();
  });

  it("re-enabling fires immediately without a confirmation dialog", () => {
    renderToggle(false);

    fireEvent.click(screen.getByRole("switch"));
    expect(mutateAsyncMock).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 42,
      enabled: true,
    });
    expect(screen.queryByText("Tạm ngừng ứng lương?")).not.toBeInTheDocument();
  });

  it("stops click propagation so tappable rows do not fire", () => {
    const parentClick = vi.fn();
    renderToggle(true, parentClick);

    fireEvent.click(screen.getByRole("switch"));

    expect(parentClick).not.toHaveBeenCalled();
  });
  it("contains a failed immediate enable without navigating the parent card", async () => {
    mutateAsyncMock.mockRejectedValueOnce(new Error("network unavailable"));
    const parentClick = vi.fn();
    renderToggle(false, parentClick);
    fireEvent.click(screen.getByRole("switch", { name: "Cho phép ứng lương" }));
    await waitFor(() => expect(mutateAsyncMock).toHaveBeenCalledTimes(1));
    expect(parentClick).not.toHaveBeenCalled();
    expect(screen.getByRole("switch")).not.toBeChecked();
  });

  it("does not let keyboard activation bubble to a containing card", () => {
    const parentKeyDown = vi.fn();
    render(
      <div onKeyDown={parentKeyDown}>
        <AdvanceRequestToggle projectId={7} employeeId={42} employeeName="Nguyễn An" enabled />
      </div>,
    );
    fireEvent.keyDown(screen.getByRole("switch", { name: "Cho phép ứng lương cho Nguyễn An" }), { key: " " });
    expect(parentKeyDown).not.toHaveBeenCalled();
  });

});
