/**
 * Type definitions for the Wallet Bulk Transfer Pipeline.
 * Mirrors the backend Go structs in
 *   backend/internal/app/services/wallet_bulk/types.go
 *   backend/internal/domain/bulk_transfer_batch.go
 *   backend/internal/domain/transactions/wallet_payment.go
 *
 * Status enums are string-literal unions so TypeScript exhaustiveness
 * checking catches missing cases at compile time.
 */

export type BulkTransferBatchStatus =
  | 'pending'
  | 'processing'
  | 'completing'
  | 'completed'
  | 'failed';

export type BulkTransferBatchEnqueueState = 'pending' | 'enqueued';

export type WalletPaymentStatus =
  | 'pending'
  | 'verified'
  | 'authorised'
  | 'completed'
  | 'failed'
  | 'reversed';

/** One parsed row of the uploaded .xlsx (mirror of BulkTransferRow in Go). */
export interface BulkTransferRowDTO {
  order_no: number;
  account_no: string;
  account_name: string;
  bank: string;
  swift_code: string;
  amount: number;
  payment_detail: string;
  vfic_code: string;
}

/** Conflict reported when uploading a file whose VFICs collide with in-flight rows. */
export interface DuplicateConflict {
  vfic_code: string;
  existing_id: number;
  status: WalletPaymentStatus;
}

/** POST /wallet/bulk-transfer/upload response (HTTP 202). */
export interface WalletBulkUploadResponse {
  batch_id: number;
  total_count: number;
  transfer_amount: number;
  estimated_fee_total: number;
  estimated_fee_per_row?: number;
  /**
   * False when the fee schedule lookup failed at upload time. When false,
   * every row will terminal-fail with ErrFeeResolution in the worker —
   * admin should fix the fee schedule before re-uploading. UI surfaces a
   * warning so admin knows the upload will not produce successful transfers.
   */
  fee_resolution_ok?: boolean;
}

/** One wallet_payment row linked to a batch (subset of fields the UI needs). */
export interface WalletBulkPaymentRow {
  id: number;
  request_id: string;
  invoice_no: string | null;
  recipient_name: string;
  recipient_account_no: string;
  recipient_bank: string;
  requested_amount: number;
  fee: number;
  status: WalletPaymentStatus;
  error_message: string | null;
  bulk_transfer_order: number | null;
  created_at: string;
  settled_at: string | null;
}

export type WalletBulkKQScope = 'all' | 'successful';

/** GET /wallet/bulk-transfer/batches/:id response. */
export interface WalletBulkBatchDetail {
  id: number;
  filename: string;
  content_hash: string;
  source: string;
  status: BulkTransferBatchStatus;
  enqueue_state: BulkTransferBatchEnqueueState;
  total_count: number;
  success_count: number;
  failed_count: number;
  transfer_amount: number;
  total_fee: number;
  ledger_txn_id: number | null;
  fee_booked_at: string | null;
  asset_id: number | null;
  created_by: number;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
  rows: WalletBulkPaymentRow[];
}

/** GET /wallet/bulk-transfer/batches response (paginated). */
export interface WalletBulkBatchListResponse {
  batches: Omit<WalletBulkBatchDetail, 'rows'>[];
  total: number;
  page: number;
  page_size: number;
}

/** Error shapes returned by the upload endpoint (subset; see handler for full list). */
export interface WalletBulkUploadError {
  error: string;
  message?: string;
  max_bytes?: number;
  max?: number;
  conflicts?: DuplicateConflict[];
  existing_batch_id?: number;
}

/** Vietnamese display label for a batch status (used in Badges + toasts). */
export function batchStatusToVietnamese(s: BulkTransferBatchStatus): string {
  switch (s) {
    case 'pending': return 'Chờ xử lý';
    case 'processing': return 'Đang xử lý';
    case 'completing': return 'Đang ghi sổ';
    case 'completed': return 'Hoàn tất';
    case 'failed': return 'Thất bại';
  }
}

/** Vietnamese display label for a wallet_payment status. */
export function paymentStatusToVietnamese(s: WalletPaymentStatus): string {
  switch (s) {
    case 'pending':
    case 'verified':
    case 'authorised':
      return 'Đang xử lý';
    case 'completed': return 'Thành công';
    case 'failed': return 'Thất bại';
    case 'reversed': return 'Đã hoàn';
  }
}

/**
 * Returns true when a completed row's invoice_no is still the VFIC code,
 * meaning the FT number hasn't yet arrived via IPN. The KQ Excel + UI
 * both surface "Đang chờ FT" in this case.
 */
export function isFTPending(row: WalletBulkPaymentRow): boolean {
  if (row.status !== 'completed') return false;
  return !row.invoice_no || row.invoice_no.startsWith('VFIC');
}
