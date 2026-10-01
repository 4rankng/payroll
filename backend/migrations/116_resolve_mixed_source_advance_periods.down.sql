-- No-op by design.
--
-- This migration only clears stale provenance: the rows it touched kept their
-- max_adv_amount, because the check-in credit had already overwritten the
-- workbook amount (see the up migration). The asset id that stamped the row is
-- still recorded on the assets table and in the audit log, so nothing is lost
-- for forensics — but there is no trustworthy per-row value to put back, and
-- guessing one would invent a provenance the data no longer supports.
SELECT 1;
