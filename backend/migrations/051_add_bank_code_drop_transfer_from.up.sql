-- Add bank_code column to banks table
ALTER TABLE banks ADD COLUMN bank_code VARCHAR(10) NOT NULL DEFAULT '' AFTER bin;

-- Backfill bank_code based on existing records
UPDATE banks SET bank_code = 'MB'   WHERE id = 4488;
UPDATE banks SET bank_code = 'VARB' WHERE id = 4489;
UPDATE banks SET bank_code = 'VCB'  WHERE id = 4490;
UPDATE banks SET bank_code = 'BIDV' WHERE id = 4491;
UPDATE banks SET bank_code = 'CTG'  WHERE id = 4492;
UPDATE banks SET bank_code = 'VPB'  WHERE id = 4493;
UPDATE banks SET bank_code = 'VIB'  WHERE id = 4494;
UPDATE banks SET bank_code = 'EIB'  WHERE id = 4495;
UPDATE banks SET bank_code = 'SHB'  WHERE id = 4496;
UPDATE banks SET bank_code = 'TPB'  WHERE id = 4497;
UPDATE banks SET bank_code = 'TCB'  WHERE id = 4498;
UPDATE banks SET bank_code = 'MSB'  WHERE id = 4499;
UPDATE banks SET bank_code = 'LPB'  WHERE id = 4500;
UPDATE banks SET bank_code = 'DAB'  WHERE id = 4501;
UPDATE banks SET bank_code = 'NASB' WHERE id = 4502;
UPDATE banks SET bank_code = 'SGB'  WHERE id = 4503;
UPDATE banks SET bank_code = 'VB'   WHERE id = 4504;
UPDATE banks SET bank_code = 'VCCB' WHERE id = 4505;
UPDATE banks SET bank_code = 'KLB'  WHERE id = 4506;
UPDATE banks SET bank_code = 'PGB'  WHERE id = 4507;
UPDATE banks SET bank_code = 'PVCB' WHERE id = 4508;
UPDATE banks SET bank_code = 'ACB'  WHERE id = 4509;
UPDATE banks SET bank_code = 'NAB'  WHERE id = 4510;
UPDATE banks SET bank_code = 'SCB'  WHERE id = 4511;
UPDATE banks SET bank_code = 'SEAB' WHERE id = 4512;
UPDATE banks SET bank_code = 'OCB'  WHERE id = 4513;
UPDATE banks SET bank_code = 'VAB'  WHERE id = 4514;
UPDATE banks SET bank_code = 'NVB'  WHERE id = 4515;
UPDATE banks SET bank_code = 'PBVN' WHERE id = 4516;
UPDATE banks SET bank_code = 'BVB'  WHERE id = 4517;
UPDATE banks SET bank_code = 'HDB'  WHERE id = 4519;
UPDATE banks SET bank_code = 'GPB'  WHERE id = 4520;
UPDATE banks SET bank_code = 'STB'  WHERE id = 4521;
UPDATE banks SET bank_code = 'ABB'  WHERE id = 4522;
UPDATE banks SET bank_code = 'HLB'  WHERE id = 4523;
UPDATE banks SET bank_code = 'SVB'  WHERE id = 4524;
UPDATE banks SET bank_code = 'VRB'  WHERE id = 4525;
UPDATE banks SET bank_code = 'UOB'  WHERE id = 4527;
UPDATE banks SET bank_code = 'WOO'  WHERE id = 4528;
UPDATE banks SET bank_code = 'IVB'  WHERE id = 4529;
UPDATE banks SET bank_code = 'VPB'  WHERE id = 4530;
UPDATE banks SET bank_code = 'VPB'  WHERE id = 4531;

ALTER TABLE banks
   DROP COLUMN transfer_from,
  DROP COLUMN deleted_at,
  DROP COLUMN created_at,
  DROP COLUMN updated_at;
