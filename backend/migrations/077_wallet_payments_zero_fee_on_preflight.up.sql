-- Zero the stamped fee on transfers that never reached the provider's
-- transfer endpoint, so SUM(fee) reflects money actually spent/lost.
--
-- The fee column is stamped at INSERT from the fee schedule for every
-- attempt, but OnePay only charges the per-transfer fee when the transfer
-- endpoint (PUT /funds_transfers) is actually called. Pre-flight
-- validation failures (amount_below_min, name_mismatch, ...) never reach
-- that endpoint and cost nothing — yet their row still shows the
-- scheduled fee. Zero those rows so `fee` means "fee actually charged",
-- not "fee that would have been charged".
--
-- OnePay response codes are numeric ("00".."90"); local pre-flight
-- rejections use alphabetic codes. A failed row whose error_code is
-- non-numeric (or missing) never reached the endpoint. Completed rows
-- and numeric-code failures keep their fee — the endpoint was called.
--
-- This is best-effort for historical rows. Going forward the disbursement
-- worker / manual handler zeroes the fee at pre-flight-rejection time via
-- WalletPaymentService.syncPatch (SyncResult.FeeWaived).
UPDATE wallet_payments
SET fee = 0
WHERE status = 'failed'
  AND (error_code IS NULL
       OR error_code NOT REGEXP '^[0-9]+$');
