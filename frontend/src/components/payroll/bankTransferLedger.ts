// Flat ledger projection of the bank-transfer history (phẳng hóa sổ cái).
// Business/data logic lives here per the .ts = logic / .tsx = rendering rule.
import type { BankTransferHistory, BankTransferHistoryTransfer } from '@/types/api/payroll.types';

/**
 * One ledger line: a single bank posting plus the record context it belongs to.
 * The flat model shows every posting as its own table row — nothing hidden
 * behind an accordion — with the record identity rendered once per record.
 */
export interface LedgerRow {
  key: string;
  record: BankTransferHistory;
  /** null when the record predates per-posting tracking (legacy data). */
  posting: BankTransferHistoryTransfer | null;
  /** True on the row that anchors the record identity cells (rowSpan anchor). */
  isFirstOfRecord: boolean;
  isLastOfRecord: boolean;
  /** How many rows the record's identity cells span. */
  recordRowCount: number;
  /** 1-based ledger line number shown in the STT column. */
  lineNumber: number;
}

/**
 * Flattens records into one row per bank posting.
 * A record with no postings still yields one row so no employee disappears
 * from the ledger.
 */
export function buildLedgerRows(records: BankTransferHistory[]): LedgerRow[] {
  const rows: LedgerRow[] = [];

  for (const record of records) {
    const postings: (BankTransferHistoryTransfer | null)[] =
      record.transfers.length > 0 ? record.transfers : [null];

    postings.forEach((posting, index) => {
      rows.push({
        key: `${record.employee_id}-${record.work_month}-${record.cycle}-${index}`,
        record,
        posting,
        isFirstOfRecord: index === 0,
        isLastOfRecord: index === postings.length - 1,
        recordRowCount: postings.length,
        lineNumber: rows.length + 1,
      });
    });
  }

  return rows;
}

export type LedgerColumnId =
  | 'stt'
  | 'employee'
  | 'period'
  | 'paymentDate'
  | 'transferCode'
  | 'bankReference'
  | 'processedAt'
  | 'amount'
  | 'recordTotal';

export interface LedgerColumn {
  id: LedgerColumnId;
  label: string;
  /** Width for the fixed-layout <colgroup>; the employee column flexes. */
  width: string;
  /** Alignment + numeric styling applied to both header and body cells. */
  alignRight?: boolean;
}

/**
 * Column order is the contract between <colgroup>, the header row and every
 * body row — cells are rendered in exactly this order. Widths budget for the
 * narrowest table breakpoint (xl=1280 with sidebar): fixed columns total
 * ~762px, leaving the rest for the employee column.
 */
export const LEDGER_COLUMNS: LedgerColumn[] = [
  { id: 'stt', label: 'STT', width: '36px' },
  { id: 'employee', label: 'Nhân viên', width: 'auto' },
  { id: 'period', label: 'Kỳ thanh toán', width: '112px' },
  { id: 'paymentDate', label: 'Ngày thanh toán', width: '90px' },
  { id: 'transferCode', label: 'Ghi chú CK', width: '96px' },
  { id: 'bankReference', label: 'Mã GD ngân hàng', width: '124px' },
  { id: 'processedAt', label: 'Thời gian xử lý', width: '88px' },
  { id: 'amount', label: 'Số tiền', width: '108px', alignRight: true },
  { id: 'recordTotal', label: 'Thực nhận (kỳ)', width: '108px', alignRight: true },
];
