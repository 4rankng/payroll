INSERT INTO transactions (description, transaction_type, amount, party, status, asset_id, created_by,
 transaction_code, settled_amount, created_at, updated_at)
 SELECT CONCAT('Trả lương cho VFIC Manpower, file: ', a.filename), 'revenue',
   (SELECT SUM(revenue_receivable) FROM timesheets WHERE id BETWEEN 12803 AND 12865),
   'VFIC Manpower', 'pending', 124, 2, UUID(), 0, '2026-05-18 00:00:00', '2026-05-18 00:00:00'
 FROM assets a WHERE a.id = 124;

 SET @txn_id = LAST_INSERT_ID();

 UPDATE timesheets SET transaction_id = @txn_id WHERE id BETWEEN 12803 AND 12865;

 INSERT INTO ledger_entries (date, account, party, debit, credit, balance, asset_id, transaction_id, created_by,
 created_at, updated_at) VALUES
   ('2026-05-18', 'cash',       'Nhân viên',     0,        20405000, 0, NULL, @txn_id, 2, NOW(), NOW()),
   ('2026-05-18', 'revenue',    'VFIC Manpower', 20405000, 0,        0, NULL, @txn_id, 2, NOW(), NOW()),
   ('2026-05-18', 'receivable', 'VFIC Manpower', 20813100, 0,        0, 124,  @txn_id, 2, NOW(), NOW()),
   ('2026-05-18', 'revenue',    'VFIC Manpower', 0,        20813100, 0, 124,  @txn_id, 2, NOW(), NOW());

 SELECT id, amount, status, asset_id FROM transactions WHERE asset_id = 124;
 SELECT COUNT(*), transaction_id FROM timesheets WHERE id BETWEEN 12803 AND 12865 GROUP BY transaction_id;
 SELECT id, account, debit, credit, asset_id FROM ledger_entries WHERE transaction_id = @txn_id ORDER BY id;
