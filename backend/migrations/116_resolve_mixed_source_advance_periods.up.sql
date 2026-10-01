-- Clear stale workbook provenance from advance periods that check-in earnings
-- have already claimed in full.
--
-- What the data actually shows
-- ---------------------------
-- The two pipelines do not add up: BatchUpsert sets max_adv_amount to the
-- workbook amount, and AccumulateSalary REPLACES it with
-- FLOOR((salary + earning) * percent / 100). So on a period that received
-- both, the last check-out credit overwrote the workbook cap instead of
-- stacking on top of it — the row holds a purely check-in-derived cap while
-- still carrying last_applied_asset_id from the earlier upload.
--
-- Verified in production: all 58 such rows have max_adv_amount = salary
-- (i.e. credited while the advance percentage was 100), and none of them
-- equals floor(salary * 70 / 100). Every check-in-only row created since the
-- percentage became configurable follows 70%.
--
-- Consequence: there is no "workbook part" left in these numbers to remove.
-- The check-in side already won on every one of them. The only thing left
-- wrong is the stale last_applied_asset_id, which makes the request backstop
-- (added alongside this migration's code) treat a clean check-in period as a
-- mixed one and refuse it.
--
-- So this migration clears the provenance and nothing else. max_adv_amount is
-- deliberately NOT rewritten: lowering these caps is a policy decision about
-- money already earned and often already paid (25 of these rows carry
-- completed requests), not a repair of a mixed-source defect. Recomputing them
-- at the current percentage is available separately through
-- RecomputeActiveCheckInMaxAdvance if the admin wants it.
--
-- Safe to re-run: it only touches rows still carrying both signatures.

UPDATE advance_payments
SET last_applied_asset_id = NULL
WHERE salary > 0
  AND last_applied_asset_id IS NOT NULL;

-- Verify: no period may claim a workbook contribution it no longer has, and
-- every check-in row must equal one of the percentages it was actually
-- credited at (100% historically, the configured one since).
SELECT
    (SELECT COUNT(*) FROM advance_payments
      WHERE salary > 0 AND last_applied_asset_id IS NOT NULL) AS remaining_mixed_rows,
    (SELECT COUNT(*) FROM advance_payments WHERE salary > 0) AS check_in_rows,
    (SELECT COUNT(*) FROM advance_payments
      WHERE salary > 0
        AND max_adv_amount NOT IN (salary, FLOOR(salary * 70 / 100))) AS unexpected_caps;
