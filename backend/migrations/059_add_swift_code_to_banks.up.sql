-- 059_add_swift_code_to_banks.up.sql
ALTER TABLE banks ADD COLUMN swift_code VARCHAR(11) NULL AFTER bank_code;

UPDATE banks SET swift_code = 'VBAVTNVX' WHERE bank_code = 'VARB';
UPDATE banks SET swift_code = 'BIDVVNVX' WHERE bank_code = 'BIDV';
UPDATE banks SET swift_code = 'ICBKVNVX' WHERE bank_code = 'CTG';
UPDATE banks SET swift_code = 'BFTVVNVX' WHERE bank_code = 'VCB';
UPDATE banks SET swift_code = 'ACBAVNVX' WHERE bank_code = 'ACB';
UPDATE banks SET swift_code = 'MBBEVNVX' WHERE bank_code = 'MB';
UPDATE banks SET swift_code = 'VTCBVNVX' WHERE bank_code = 'TCB';
UPDATE banks SET swift_code = 'STCBVNVX' WHERE bank_code = 'STB';
UPDATE banks SET swift_code = 'VIBVNVX'  WHERE bank_code = 'VIB';
UPDATE banks SET swift_code = 'VPBKVNVX' WHERE bank_code = 'VPB';
UPDATE banks SET swift_code = 'EIBKVNVX' WHERE bank_code = 'EIB';
UPDATE banks SET swift_code = 'SHBKVNVX' WHERE bank_code = 'SHB';
UPDATE banks SET swift_code = 'TPBCVNVX' WHERE bank_code = 'TPB';
UPDATE banks SET swift_code = 'SEABVNVX' WHERE bank_code = 'SEAB';
UPDATE banks SET swift_code = 'OCBNVNVX' WHERE bank_code = 'OCB';
UPDATE banks SET swift_code = 'MSBKVNVX' WHERE bank_code = 'MSB';
UPDATE banks SET swift_code = 'LPBKVNVX' WHERE bank_code = 'LPB';
UPDATE banks SET swift_code = 'HDBKVNVX' WHERE bank_code = 'HDB';
UPDATE banks SET swift_code = 'NABKVNVX' WHERE bank_code = 'NAB';
UPDATE banks SET swift_code = 'NASBVNVX' WHERE bank_code = 'NASB';
-- Update SWIFT codes for Vietnamese banks with NULL swift_code
UPDATE banks SET swift_code = 'EACBVNVX' WHERE id = 4501; -- Đông Á (DAB)
UPDATE banks SET swift_code = 'SGTTVNVX' WHERE id = 4503; -- Sài Gòn Công thương (SGB)
UPDATE banks SET swift_code = 'VNTTVNVX' WHERE id = 4504; -- Việt Nam Thương tín (VIETBANK)
UPDATE banks SET swift_code = 'BVBVVNVX' WHERE id = 4505; -- BVBank / Bản Việt (VCCB)
UPDATE banks SET swift_code = 'KLBKVNVX' WHERE id = 4506; -- Kiên Long (KLB)
UPDATE banks SET swift_code = 'PGBLVNVX' WHERE id = 4507; -- PGBank (PGB)
UPDATE banks SET swift_code = 'WBVNVNVX' WHERE id = 4508; -- PVcomBank (PVCB)
UPDATE banks SET swift_code = 'SACLVNVX' WHERE id = 4511; -- Sài Gòn (SCB)
UPDATE banks SET swift_code = 'JACBVNVX' WHERE id = 4514; -- Việt Á (VAB)
UPDATE banks SET swift_code = 'NCBIVNVX' WHERE id = 4515; -- Quốc Dân (NCB/NVB)
UPDATE banks SET swift_code = 'VIDPVNVX' WHERE id = 4516; -- VID Public Bank
UPDATE banks SET swift_code = 'BVBVVNVX' WHERE id = 4517; -- Bảo Việt (BVB)
UPDATE banks SET swift_code = 'MHBVVNVX' WHERE id = 4518; -- Việt Nam Hiện Đại (MBV)
UPDATE banks SET swift_code = 'GBNKVNVX' WHERE id = 4520; -- Dầu khí toàn cầu (GPB)
UPDATE banks SET swift_code = 'ABBKVNVX' WHERE id = 4522; -- An Bình (ABBANK)
UPDATE banks SET swift_code = 'HLBBVNVX' WHERE id = 4523; -- Hong Leong VN (HLB)
UPDATE banks SET swift_code = 'SHBKVNVX' WHERE id = 4524; -- Shinhan Việt Nam (SHBVN)
UPDATE banks SET swift_code = 'VRBBVNVX' WHERE id = 4525; -- Việt Nga (VRB)
UPDATE banks SET swift_code = 'CBBVVNVX' WHERE id = 4526; -- Xây dựng Việt Nam (CBB)
UPDATE banks SET swift_code = 'UOVBVNVX' WHERE id = 4527; -- United Overseas Bank VN (UOB)
UPDATE banks SET swift_code = 'HVBKVNVX' WHERE id = 4528; -- Woori Bank VN
UPDATE banks SET swift_code = 'IABBVNVX' WHERE id = 4529; -- Indovina (IVB)
