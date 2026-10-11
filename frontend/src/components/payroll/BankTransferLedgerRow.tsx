import { Badge } from '@/components/ui/badge';
import { TableCell, TableRow } from '@/components/ui/table';
import type { BankTransferHistory } from '@/types/api/payroll.types';
import {
  formatBankTransferDate,
  formatBankTransferDateTime,
  formatBankTransferPeriod,
  formatBankTransferTime,
} from '@/utils/bankTransferHistoryHelpers';
import { formatCurrency } from '@/utils/formatters';
import type { LedgerRow } from './bankTransferLedger';

// Rendering-only: all flattening/labels live in ./bankTransferLedger(.ts).

const NUMERIC_CELL = 'font-financial text-[11px] font-semibold tabular-nums';
const HEADER_CELL =
  'h-9 px-2 whitespace-nowrap text-left align-middle font-display text-[11px] font-bold uppercase tracking-[0.06em] text-fg-tertiary';

/** Identity cells span every posting row of their record (flat ledger model). */
const IDENTITY_CELL = 'align-top';

function CellShell({
  children,
  right = false,
  className = '',
}: {
  children: React.ReactNode;
  right?: boolean;
  className?: string;
}) {
  return (
    <TableCell className={`py-1.5 ${right ? 'pl-2 pr-3 text-right' : 'px-2'} ${className}`}>
      {children}
    </TableCell>
  );
}

/**
 * One ledger line. The record identity (employee, period, payment date, total)
 * renders once per record via rowSpan; continuation rows carry only posting
 * cells so every bank posting is visible without expanding anything.
 */
export function LedgerTableRow({ row }: { row: LedgerRow }) {
  const { record, posting, isFirstOfRecord, recordRowCount, lineNumber } = row;
  const empty = posting === null;

  return (
    <TableRow
      className={`h-10 border-b ${
        row.isLastOfRecord ? 'border-utility-gray-300' : 'border-utility-gray-200'
      }`}
    >
      <CellShell className={`${NUMERIC_CELL} text-fg-tertiary`}>
        {String(lineNumber).padStart(2, '0')}
      </CellShell>

      {isFirstOfRecord && (
        <>
          <TableCell rowSpan={recordRowCount} className={`px-2 py-1.5 ${IDENTITY_CELL}`}>
            <p className="break-words font-display text-xs font-bold leading-tight text-fg-primary">
              {record.employee_name}
            </p>
            <p className="mt-0.5 break-words text-[11px] leading-tight text-fg-tertiary">
              CCCD {record.employee_cccd || '—'}
              {record.project_names.length > 0 && (
                <span title={record.project_names.join(', ')}>
                  {' · '}
                  {record.project_names.join(', ')}
                </span>
              )}
            </p>
          </TableCell>
          <TableCell rowSpan={recordRowCount} className={`px-2 py-1.5 ${IDENTITY_CELL}`}>
            <p
              className={`${NUMERIC_CELL} whitespace-nowrap text-fg-primary`}
              aria-label={`Từ ${formatBankTransferDate(record.from_date)} đến ${formatBankTransferDate(record.to_date)}`}
            >
              {formatBankTransferPeriod(record.from_date, record.to_date)}
            </p>
            <Badge className="mt-0.5 rounded-full border border-utility-gray-300 bg-utility-gray-50 px-1.5 py-0 font-display text-[10px] font-bold text-fg-secondary hover:bg-utility-gray-50">
              Kỳ {record.cycle}
            </Badge>
          </TableCell>
          <TableCell rowSpan={recordRowCount} className={`px-2 py-1.5 ${IDENTITY_CELL}`}>
            <time
              dateTime={record.payment_date}
              className={`${NUMERIC_CELL} whitespace-nowrap text-fg-primary`}
            >
              {formatBankTransferDate(record.payment_date)}
            </time>
          </TableCell>
        </>
      )}

      <CellShell className={`${NUMERIC_CELL} break-all text-fg-primary`}>
        {empty ? '—' : posting.transfer_code || '—'}
      </CellShell>
      <CellShell className={`${NUMERIC_CELL} break-all text-fg-primary`}>
        {empty || !posting.bank_reference ? '—' : posting.bank_reference}
      </CellShell>
      <CellShell className={`${NUMERIC_CELL} whitespace-nowrap text-fg-secondary`}>
        {empty || !posting.paid_at ? (
          '—'
        ) : (
          <time dateTime={posting.paid_at} className="block">
            <span className="block">{formatBankTransferDate(posting.paid_at.slice(0, 10))}</span>
            <span className="block text-fg-tertiary">{formatBankTransferTime(posting.paid_at)}</span>
          </time>
        )}
      </CellShell>
      <CellShell right className={`${NUMERIC_CELL} whitespace-nowrap text-fg-primary`}>
        {empty ? '—' : formatCurrency(posting.amount)}
      </CellShell>

      {isFirstOfRecord && (
        <TableCell
          rowSpan={recordRowCount}
          className={`py-1.5 pl-2 pr-3 text-right ${IDENTITY_CELL}`}
        >
          <span className="font-financial text-xs font-bold tabular-nums whitespace-nowrap text-emerald-700">
            {formatCurrency(record.total_amount)}
          </span>
        </TableCell>
      )}
    </TableRow>
  );
}

/**
 * Dense mobile record block: identity line + metadata line + one line per
 * bank posting. Everything visible, nothing to tap open.
 */
export function LedgerMobileRecord({ record }: { record: BankTransferHistory }) {
  const postings = record.transfers.length > 0 ? record.transfers : [null];

  return (
    <div className="border-b border-utility-gray-300 px-3 py-2 last:border-b-0">
      <div className="flex items-baseline justify-between gap-3">
        <p className="min-w-0 break-words font-display text-[13px] font-bold leading-tight text-fg-primary">
          {record.employee_name}
        </p>
        <p className="shrink-0 whitespace-nowrap font-financial text-[15px] font-bold tabular-nums leading-tight text-emerald-700">
          {formatCurrency(record.total_amount)}
        </p>
      </div>

      <p className="mt-0.5 break-words text-[11px] leading-snug text-fg-tertiary">
        CCCD {record.employee_cccd || '—'}
        {record.project_names.length > 0 && <> · {record.project_names.join(', ')}</>}
        {' · '}
        <span
          aria-label={`Từ ${formatBankTransferDate(record.from_date)} đến ${formatBankTransferDate(record.to_date)}`}
        >
          {formatBankTransferPeriod(record.from_date, record.to_date)}
        </span>
        {' · '}
        <span className="font-semibold text-fg-secondary">Kỳ {record.cycle}</span>
        {' · TT '}
        <time dateTime={record.payment_date}>{formatBankTransferDate(record.payment_date)}</time>
      </p>

      <div className="mt-1 space-y-0.5" role="list" aria-label={`${postings.length} bút toán ngân hàng`}>
        {postings.map((posting, index) => (
          <div
            key={posting ? `${posting.transfer_code}-${posting.bank_reference}-${index}` : `empty-${index}`}
            role="listitem"
            className="flex flex-wrap items-baseline gap-x-2 text-[11px] leading-snug"
          >
            <span className={`${NUMERIC_CELL} text-fg-secondary`}>
              {posting?.transfer_code || '—'}
            </span>
            <span className={`${NUMERIC_CELL} break-all text-fg-primary`}>
              {posting?.bank_reference || '—'}
            </span>
            <span className="whitespace-nowrap text-fg-tertiary">
              {posting?.paid_at ? (
                <time dateTime={posting.paid_at}>{formatBankTransferDateTime(posting.paid_at)}</time>
              ) : (
                '—'
              )}
            </span>
            <span className={`${NUMERIC_CELL} ml-auto whitespace-nowrap text-fg-primary`}>
              {posting ? formatCurrency(posting.amount) : '—'}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
