-- Migration: Fix missing transaction from BulkTransferResultParsed event
-- EventID: 9d6550a7-a39e-43b1-ae24-fac079a96e80
-- AssetID: 28, BulkFileID: 10
-- Net Amount: 3,561,250 VND | Gross Amount: 3,632,475 VND | Fee: 71,225 VND (2%)

-- Insert the missing revenue transaction
-- This transaction represents the bulk transfer payment processed on 2025-11-17
INSERT INTO transactions (
    id,
    description,
    transaction_type,
    amount,
    party,
    asset_id,
    user_id,
    created_by,
    transaction_code,
    status,
    created_at,
    updated_at
) VALUES (
    NULL,
    'Kết quả chuyển tiền theo bảng kê - 25111721240253699.xlsx',
    'revenue',
    3632475,  -- Gross amount before 2% fee
    'VFIC Manpower',
    28,
    NULL,
    2,
    UUID(),
    'pending',
    '2025-11-17 14:26:21',
    NOW()
);

-- Get the ID of the transaction we just inserted
SET @bulk_transaction_id = LAST_INSERT_ID();

-- Insert the 4 required ledger entries following the established pattern
-- 1. Receivable entry (debit gross amount)
INSERT INTO ledger_entries (
    date,
    account,
    party,
    debit,
    credit,
    balance,
    asset_id,
    transaction_id,
    created_by,
    created_at,
    updated_at
) VALUES (
    '2025-11-17',
    'receivable',
    'VFIC Manpower',
    3632475,  -- Gross amount
    0,
    3632475,
    28,
    @bulk_transaction_id,
    2,
    '2025-11-17 14:26:21',
    NOW()
);

-- 2. Revenue entry (credit gross amount)
INSERT INTO ledger_entries (
    date,
    account,
    party,
    debit,
    credit,
    balance,
    asset_id,
    transaction_id,
    created_by,
    created_at,
    updated_at
) VALUES (
    '2025-11-17',
    'revenue',
    'VFIC Manpower',
    0,
    3632475,  -- Gross amount
    -3632475,
    28,
    @bulk_transaction_id,
    2,
    '2025-11-17 14:26:21',
    NOW()
);

-- 3. Cash entry (credit net amount to employees)
INSERT INTO ledger_entries (
    date,
    account,
    party,
    debit,
    credit,
    balance,
    asset_id,
    transaction_id,
    created_by,
    created_at,
    updated_at
) VALUES (
    '2025-11-17',
    'cash',
    'Nhân viên',
    0,
    3561250,  -- Net amount paid to employees
    -3561250,
    28,
    @bulk_transaction_id,
    2,
    '2025-11-17 14:26:21',
    NOW()
);

-- 4. Revenue offset entry (debit net amount)
INSERT INTO ledger_entries (
    date,
    account,
    party,
    debit,
    credit,
    balance,
    asset_id,
    transaction_id,
    created_by,
    created_at,
    updated_at
) VALUES (
    '2025-11-17',
    'revenue',
    'VFIC Manpower',
    3561250,  -- Net amount
    0,
    -3561250,
    28,
    @bulk_transaction_id,
    2,
    '2025-11-17 14:26:21',
    NOW()
);
