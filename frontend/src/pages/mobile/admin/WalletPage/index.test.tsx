import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import WalletPageMobile from "./index";

const bulkTransferBatchListProps = vi.fn();
const employeeAccountLookupDialogProps = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: {
      available: 12000000,
      pending_out: 3000000,
      as_of: "2026-07-25T08:00:00+07:00",
    },
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/components/wallet/WalletTransactionsList", () => ({
  default: () => <div>Danh sách giao dịch</div>,
}));

vi.mock("@/components/wallet/CreateManualDisbursementDialog", () => ({
  default: () => null,
}));

vi.mock('@/components/wallet/EmployeeAccountLookupDialog', () => ({
  EmployeeAccountLookupDialog: (props: { open: boolean }) => {
    employeeAccountLookupDialogProps(props);
    return props.open ? <div>Hộp thoại tra cứu tài khoản</div> : null;
  },
}));

vi.mock("@/components/wallet/BulkTransferBatchList", () => ({
  BulkTransferBatchList: (props: {
    selectedBatchId?: number | null;
    onSelectedBatchIdChange?: (batchId: number | null) => void;
  }) => {
    bulkTransferBatchListProps(props);
    return <div>Tiến độ chuyển lô</div>;
  },
}));

vi.mock("@/components/wallet/BulkTransferUploadDialog", () => ({
  BulkTransferUploadDialog: ({
    open,
    onViewProgress,
  }: {
    open: boolean;
    onViewProgress?: (batchId: number) => void;
  }) =>
    open ? (
      <div>
        Hộp thoại tải file chuyển lô
        <button type="button" onClick={() => onViewProgress?.(42)}>
          Xem tiến độ
        </button>
      </div>
    ) : null,
}));

describe("WalletPageMobile", () => {
  it("exposes bulk-transfer upload and progress beside existing wallet actions", () => {
    render(<WalletPageMobile />);

    expect(screen.getByText("Tiến độ chuyển lô")).toBeInTheDocument();

    const uploadButton = screen.getByRole("button", { name: "Tải file" });
    expect(uploadButton).toHaveClass("min-h-11");
    fireEvent.click(uploadButton);

    expect(screen.getByText("Hộp thoại tải file chuyển lô")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Xem tiến độ" }));

    expect(bulkTransferBatchListProps).toHaveBeenLastCalledWith(
      expect.objectContaining({
        selectedBatchId: 42,
        onSelectedBatchIdChange: expect.any(Function),
      }),
    );
    expect(screen.getByRole("button", { name: "Chuyển tiền" })).toHaveClass(
      "min-h-11",
    );

    const lookupButton = screen.getByRole('button', { name: 'Tra cứu tài khoản' });
    expect(lookupButton).toHaveClass('min-h-11');
    fireEvent.click(lookupButton);
    expect(screen.getByText('Hộp thoại tra cứu tài khoản')).toBeInTheDocument();
    expect(employeeAccountLookupDialogProps).toHaveBeenLastCalledWith(
      expect.objectContaining({ open: true }),
    );
  });
});
