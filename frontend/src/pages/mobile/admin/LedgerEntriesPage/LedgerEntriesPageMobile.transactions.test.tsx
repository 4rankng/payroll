import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import LedgerEntriesPageMobile from "./index";

const mocks = vi.hoisted(() => ({
  transactionsCalls: [] as Record<string, unknown>[],
  ledgerEntriesCalls: [] as Record<string, unknown>[],
  openTransactionDetails: vi.fn(),
  openAddLedgerEntry: vi.fn(),
  headerProps: null as Record<string, unknown> | null,
  listProps: null as Record<string, unknown> | null,
}));

const transaction = {
  id: 4242,
  description: "Chi lương tuần",
  transaction_code: "itest2026102842426746400000000000000000_vendor",
  transaction_type: "payroll_payment",
  amount: 1_000_000,
  party: "itest2026102842426746400000000000000000_vendor",
  status: "pending",
  settlement_method: "payable",
  created_by: 9,
  created_at: "2026-10-28T10:00:00Z",
  updated_at: "2026-10-28T11:00:00Z",
};

vi.mock("@/hooks/transactions/useTransactions", () => ({
  useTransactions: (filters: Record<string, unknown>) => {
    mocks.transactionsCalls.push(filters);
    return {
      data: {
        data: [transaction],
        pagination: { page: filters.page, pageSize: 50, totalPages: 1, totalRecords: 1 },
      },
      isLoading: false,
    };
  },
}));
vi.mock("@/hooks/ledger/useLedgerEntries", () => ({
  useLedgerEntries: (filters: Record<string, unknown>) => {
    mocks.ledgerEntriesCalls.push(filters);
    return { data: undefined, isLoading: false };
  },
  useLedgerEntry: () => ({ data: undefined }),
  useReverseLedgerEntry: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock("@/hooks/ledger/useLedgerBalance", () => ({
  useOverallBalance: () => ({ data: 0, isLoading: false }),
  useLedgerSummary: () => ({ data: undefined, isLoading: false }),
}));
vi.mock("@/hooks/ledger/useLedgerManagement", () => ({
  useLedgerManagement: () => ({
    importOnePayFeeReport: vi.fn(),
    isImportingOnePayFeeReport: false,
    recalculateBalances: vi.fn(),
    runWalletSettlement: vi.fn(),
    isRecalculating: false,
    isRunningWalletSettlement: false,
  }),
}));
vi.mock("@/hooks/ledger/useLedgerMetadata", () => ({
  useLedgerMetadata: () => ({ data: [], isLoading: false }),
}));
vi.mock("@/hooks/api/usePayrolls", () => ({
  useSendPayrollReportEmail: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock("@/hooks/api/useAdvancePaymentReconciliation", () => ({
  useSendReconciliationEmail: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock("@/hooks/api/useProjects", () => ({
  useProjects: () => ({ data: { data: [] } }),
}));
vi.mock("@/hooks/useModalNavigation", () => ({
  useGeneralModals: () => ({
    openTransactionDetails: mocks.openTransactionDetails,
    openAddLedgerEntry: mocks.openAddLedgerEntry,
  }),
}));
vi.mock("@/components/ledger/LedgerPageHeader", () => ({
  LedgerPageHeader: (props: Record<string, unknown>) => {
    mocks.headerProps = props;
    return (
      <div>
        <button type="button" onClick={() => (props.onExportSaoKePayroll as () => void)()}>
          act-export-payroll
        </button>
        <button type="button" onClick={() => (props.onExportSaoKeAdvance as () => void)()}>
          act-export-advance
        </button>
        <button type="button" onClick={() => (props.onUploadSettlementResult as () => void)()}>
          act-settlement-upload
        </button>
        <button type="button" onClick={() => (props.onSimulateSettlement as () => void)()}>
          act-settlement-simulate
        </button>
      </div>
    );
  },
}));
vi.mock("@/components/ledger/LedgerSummaryCard", () => ({ LedgerSummaryCard: () => null }));
vi.mock("@/components/ledger/LedgerMobileList", () => ({
  LedgerMobileList: (props: Record<string, unknown>) => {
    mocks.listProps = props;
    const entries = props.entries as { id: number }[];
    return (
      <button type="button" onClick={() => (props.onRowClick as (e: unknown) => void)(entries[0])}>
        row-{entries[0]?.id}
      </button>
    );
  },
}));
vi.mock("@/components/transaction/TransactionFilters", () => ({
  TransactionFilters: (props: Record<string, unknown>) => (
    <button
      type="button"
      onClick={() =>
        (props.onFiltersChange as (f: Record<string, unknown>) => void)({
          ...(props.filters as Record<string, unknown>),
          search: "vendor",
          transaction_type: "payroll_payment",
          status: "pending",
          page: 1,
        })
      }
    >
      change-filter
    </button>
  ),
}));
vi.mock("@/components/ledger/DoubleEntryModal", () => ({ DoubleEntryModal: () => null }));
vi.mock("@/components/timesheet/PayrollReportEmailDialog", () => ({
  PayrollReportEmailDialog: () => null,
}));
vi.mock("@/components/advance-payment/AdvancePaymentEmailDialog", () => ({
  AdvancePaymentEmailDialog: () => null,
}));
vi.mock("@/components/transaction/SaoKeHistoryDialog", () => ({ SaoKeHistoryDialog: () => null }));
vi.mock("@/components/transaction/ExportSaoKeDialog", () => ({
  ExportSaoKeDialog: ({ open }: { open: boolean }) => (open ? <div>dlg-export-payroll</div> : null),
}));
vi.mock("@/components/advance-payment/AdvancePaymentExportDialog", () => ({
  AdvancePaymentExportDialog: ({ open }: { open: boolean }) =>
    open ? <div>dlg-export-advance</div> : null,
}));
vi.mock("@/components/ledger/OnePayFeeReportDialog", () => ({
  OnePayFeeReportDialog: ({ open }: { open: boolean }) => (open ? <div>dlg-onepay-fee</div> : null),
}));
vi.mock("@/components/transaction/SettlementResultUploadDialog", () => ({
  SettlementResultUploadDialog: ({ open }: { open: boolean }) =>
    open ? <div>dlg-settlement-upload</div> : null,
}));
vi.mock("@/components/ledger/SettlementSimulationDialog", () => ({
  SettlementSimulationDialog: ({ open }: { open: boolean }) =>
    open ? <div>dlg-settlement-simulate</div> : null,
}));

describe("LedgerEntriesPageMobile Transaction model (PAYR-20)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.transactionsCalls.length = 0;
    mocks.ledgerEntriesCalls.length = 0;
    mocks.headerProps = null;
    mocks.listProps = null;
  });

  it("danh sách dùng useTransactions — không dùng useLedgerEntries (một model/API)", () => {
    render(
      <MemoryRouter>
        <LedgerEntriesPageMobile />
      </MemoryRouter>,
    );
    expect(mocks.transactionsCalls.length).toBeGreaterThan(0);
    expect(mocks.ledgerEntriesCalls).toHaveLength(0);
  });

  it("filter search/type/status đi thẳng vào useTransactions", () => {
    render(
      <MemoryRouter>
        <LedgerEntriesPageMobile />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByText("change-filter"));
    const last = mocks.transactionsCalls[mocks.transactionsCalls.length - 1];
    expect(last).toMatchObject({
      search: "vendor",
      transaction_type: "payroll_payment",
      status: "pending",
    });
  });

  it("row click mở transaction details với transaction id (adapter giữ id)", () => {
    render(
      <MemoryRouter>
        <LedgerEntriesPageMobile />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByText("row-4242"));
    expect(mocks.openTransactionDetails).toHaveBeenCalledWith("4242");
  });

  it("export sao kê lương/ứng + settlement upload/mô phỏng mở dialog tương ứng", () => {
    render(
      <MemoryRouter>
        <LedgerEntriesPageMobile />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByText("act-export-payroll"));
    fireEvent.click(screen.getByText("act-export-advance"));
    fireEvent.click(screen.getByText("act-settlement-upload"));
    fireEvent.click(screen.getByText("act-settlement-simulate"));
    expect(screen.getByText("dlg-export-payroll")).toBeTruthy();
    expect(screen.getByText("dlg-export-advance")).toBeTruthy();
    expect(screen.getByText("dlg-settlement-upload")).toBeTruthy();
    expect(screen.getByText("dlg-settlement-simulate")).toBeTruthy();
  });
});
