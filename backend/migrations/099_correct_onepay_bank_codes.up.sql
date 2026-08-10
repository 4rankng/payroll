-- OnePay rejects these historical identifiers with code 12 (invalid bank code).
-- The rows are shared by existing employee records, so correcting the catalog
-- restores validation without changing employee bank details.
UPDATE banks
SET swift_code = 'VBAAVNVX'
WHERE bank_code = 'VARB' AND swift_code IN ('VBAVVNVX', 'VBAVTNVX');

UPDATE banks
SET swift_code = 'HDBCVNVX'
WHERE bank_code = 'HDB' AND swift_code = 'HDBKVNVX';
