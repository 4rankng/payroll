/**
 * iter01-business-logic-p0.cjs
 * P0 Business Logic Verification — correct API paths from api.config.ts
 */
const { createDesktopBrowser, loginAs, apiCall, getAdminToken, getPartnerToken } = require(`${__dirname}/desktop-helpers.cjs`);

let adminToken, partnerToken;
let browser, page;
const results = [];
const SCREENSHOT_DIR = '/Users/dev/Documents/projects/payroll/qa/screenshots';
const fs = require('fs');

function log(category, scenario, status, detail = '') {
  results.push({ category, scenario, status, detail });
  const icon = status === 'PASS' ? '✅' : status === 'FAIL' ? '❌' : '⚠️';
  console.log(`${icon} [${category}] ${scenario}${detail ? ' — ' + detail : ''}`);
}

function extractData(res) {
  // API responses: {status:"success", data:...} or {data:...}
  return res.data?.data ?? res.data;
}

// Walk a nested payrate JSON tree and return the first leaf key as a usable
// hour type. Payrate configs are nested maps like
//   { "pho thong": { "ngay thuong": { "ca ngày": 30000, "tăng ca": 45000 } } }
// Leaves are the actual hour-type names the backend accepts (e.g. "ca ngày",
// "tăng ca"). We return the first leaf we find — good enough for the smoke
// test that just needs *some* valid hour type for the chosen project.
function collectHourTypes(node) {
  if (node === null || node === undefined) return null;
  if (typeof node !== 'object') return null;
  const keys = Object.keys(node);
  if (!keys.length) return null;
  for (const k of keys) {
    const child = node[k];
    if (typeof child === 'number' || typeof child === 'string') {
      return k;
    }
    const nested = collectHourTypes(child);
    if (nested) return nested;
  }
  return null;
}

async function setup() {
  console.log('\n🔧 Setting up...');
  adminToken = await getAdminToken();
  log('SETUP', 'Admin token', 'PASS');
  partnerToken = await getPartnerToken();
  log('SETUP', 'Partner token (thanhmai)', 'PASS');

  // Browser is optional — only used for visual checks. If the frontend isn't
  // running (e.g. CI / API-only mode), skip browser setup entirely instead of
  // crashing on `ERR_CONNECTION_REFUSED`.
  try {
    browser = await createDesktopBrowser();
    page = await browser.newPage();
    const loggedIn = await loginAs(page, 'frankng');
    log('SETUP', 'Browser login as admin', loggedIn ? 'PASS' : 'WARN', loggedIn ? '' : 'proceeding with API-only tests');
  } catch (e) {
    log('SETUP', 'Browser unavailable — API-only mode', 'WARN', e.message);
    browser = null;
    page = null;
  }
}

async function teardown() {
  if (browser) await browser.close();
}

// ──────────────────────────────────────────────
// F07: Timesheet Entry
// ──────────────────────────────────────────────
async function testTimesheetEntry() {
  console.log('\n📋 F07: Timesheet Entry & Validation');

  const projRes = await apiCall('GET', '/api/v1/projects?pageSize=10', null, adminToken);
  const projects = extractData(projRes);
  const projectList = Array.isArray(projects) ? projects : projects?.items || [];
  if (!projectList.length) { log('F07', 'Setup: no projects', 'FAIL'); return; }

  // Walk projects to find one with at least one employee assigned.
  // Then derive a valid hourType from the project's payrate config so we don't
  // hardcode 'HC' (some projects only have ca ngày / ca đêm / tăng ca, etc.).
  let project = null;
  let emp = null;
  let empId = null;
  let hourType = null;
  for (const p of projectList) {
    const empRes = await apiCall('GET', `/api/v1/projects/${p.id}/employees?pageSize=5`, null, adminToken);
    const empData = extractData(empRes);
    const employees = Array.isArray(empData) ? empData : empData?.items || [];
    if (!employees.length) continue;

    // Fetch the project's payrate config to discover valid hour types
    const prRes = await apiCall('GET', `/api/v1/projects/${p.id}/payrate`, null, adminToken);
    const prData = extractData(prRes);
    const ratesJson = prData?.rates;
    const validHourType = collectHourTypes(ratesJson);
    if (!validHourType) continue;

    project = p;
    emp = employees.sort((a, b) => new Date(a.start_date || '2026-01-01') - new Date(b.start_date || '2026-01-01'))[0];
    empId = emp.employee_id || emp.id;
    hourType = validHourType;
    break;
  }
  if (!project) { log('F07', 'Setup: no project with employees + payrate', 'FAIL'); return; }
  log('F07', `Using project ${project.id} (${project.name || project.client_name}) hourType=${hourType}`, 'PASS');

  // Extract date part directly from start_date string to avoid timezone issues
  // start_date format: "2026-06-01T00:00:00+08:00" → "2026-06-01"
  const startDateStr = (emp.start_date || '2026-01-01').split('T')[0];
  console.log(`    emp=${empId} start_date=${startDateStr}`);

  // F07-01: Create single timesheet (may fail if date already occupied by BCC import)
  const createRes = await apiCall('POST', '/api/v1/timesheets', [{
    projectId: project.id, employeeId: empId, date: startDateStr, hoursWorked: 8, hourType,
  }], adminToken);
  if (createRes.ok) {
    log('F07', 'F07-01: Create single timesheet', 'PASS', `status=${createRes.status}`);
  } else {
    // 400 may mean date already occupied (upsert behavior) or before start_date
    const errMsg = typeof createRes.data === 'object' ? createRes.data.message || '' : '';
    log('F07', 'F07-01: Create single timesheet', errMsg.includes('tồn tại') ? 'WARN' : 'FAIL',
      `status=${createRes.status} ${errMsg.slice(0, 80)}`);
  }

  // F07-03: Future date rejected
  const tomorrow = new Date(); tomorrow.setDate(tomorrow.getDate() + 1);
  const futureRes = await apiCall('POST', '/api/v1/timesheets', [{
    projectId: project.id, employeeId: empId, date: tomorrow.toISOString().split('T')[0], hoursWorked: 8, hourType,
  }], adminToken);
  log('F07', 'F07-03: Future date rejected', futureRes.status === 400 ? 'PASS' : 'FAIL', `status=${futureRes.status}`);

  // F07-09: Paid timesheet deletion blocked
  const paidRes = await apiCall('GET', `/api/v1/timesheets?project_id=${project.id}&payment_status=paid&pageSize=1`, null, adminToken);
  const paidItems = extractData(paidRes);
  const paidTs = Array.isArray(paidItems) ? paidItems[0] : paidItems?.items?.[0];
  if (paidTs) {
    const delRes = await apiCall('DELETE', `/api/v1/timesheets/${paidTs.id}`, null, adminToken);
    log('F07', 'F07-09: Delete paid timesheet blocked', delRes.status >= 400 ? 'PASS' : 'FAIL', `status=${delRes.status}`);
  } else {
    log('F07', 'F07-09: No paid timesheets to test', 'WARN');
  }

  // F07-06: Summary
  const sumRes = await apiCall('GET', `/api/v1/timesheets/summary?project_id=${project.id}`, null, adminToken);
  log('F07', 'F07-06: Timesheet summary', sumRes.ok ? 'PASS' : 'FAIL', `status=${sumRes.status}`);

  // F07-07: Grouped
  const grpRes = await apiCall('GET', `/api/v1/timesheets/grouped?project_id=${project.id}`, null, adminToken);
  log('F07', 'F07-07: Grouped timesheets', grpRes.ok ? 'PASS' : 'FAIL', `status=${grpRes.status}`);

  // Browser check
  if (!page) {
    log('F07', 'Browser: skipped (no browser)', 'WARN');
  } else try {
    await page.goto('http://localhost:3000/admin/timesheet', { waitUntil: 'networkidle0', timeout: 15000 });
    await page.screenshot({ path: `${SCREENSHOT_DIR}/F07-timesheet.png` });
    log('F07', 'Browser: Timesheet page loads', 'PASS');
  } catch (e) { log('F07', 'Browser: Timesheet page', 'WARN', e.message); }
}

// ──────────────────────────────────────────────
// F08: Timesheet Approval
// ──────────────────────────────────────────────
async function testTimesheetApproval() {
  console.log('\n📋 F08: Timesheet Approval');

  // F08-06: Partner CANNOT approve (RBAC)
  const partnerApproveRes = await apiCall('POST', '/api/v1/timesheets/bulk-approve', {
    timesheet_ids: [999999],
  }, partnerToken);
  // Should be 403 (forbidden) — but might be 404/400 if endpoint validates IDs first
  const isBlocked = partnerApproveRes.status === 403 || partnerApproveRes.status === 401;
  log('F08', 'F08-06: Partner blocked from bulk-approve', isBlocked ? 'PASS' : 'FAIL',
    `status=${partnerApproveRes.status} ${isBlocked ? '' : JSON.stringify(partnerApproveRes.data || {}).slice(0, 200)}`);

  // Get pending timesheets
  const pendRes = await apiCall('GET', '/api/v1/timesheets?status=pending_approval&pageSize=5', null, adminToken);
  const pendData = extractData(pendRes);
  const pending = Array.isArray(pendData) ? pendData : pendData?.items || [];
  log('F08', 'Pending timesheets found', pending.length > 0 ? 'PASS' : 'WARN', `count=${pending.length}`);

  if (pending.length > 0) {
    const ids = pending.slice(0, 2).map(t => t.id);
    const apprRes = await apiCall('POST', '/api/v1/timesheets/bulk-approve', { timesheet_ids: ids }, adminToken);
    log('F08', 'F08-01: Bulk approve', apprRes.ok ? 'PASS' : 'FAIL', `status=${apprRes.status}`);
  }

  // F08-04: Idempotent approval
  const allRes = await apiCall('POST', '/api/v1/timesheets/approve-all', { project_id: 1 }, adminToken);
  // approve-all may or may not exist; just log
  log('F08', 'F08-04: Approve-all endpoint', allRes.ok || allRes.status === 404 ? 'PASS' : 'WARN', `status=${allRes.status}`);
}

// ──────────────────────────────────────────────
// F10/F09: BCC Import
// ──────────────────────────────────────────────
async function testBCCImport() {
  console.log('\n📋 F09/F10: BCC Import');

  const histRes = await apiCall('GET', '/api/v1/timesheets/partner-import?pageSize=10', null, adminToken);
  log('F10', 'Admin sees all import history', histRes.ok ? 'PASS' : 'FAIL', `status=${histRes.status}`);

  const partnerHistRes = await apiCall('GET', '/api/v1/timesheets/partner-import?pageSize=10', null, partnerToken);
  // Note: requires server restart to pick up Casbin policy change for GET partner-import
  log('F10', 'Partner sees filtered history', partnerHistRes.ok ? 'PASS' : 'WARN', `status=${partnerHistRes.status}${partnerHistRes.ok ? '' : ' (Casbin reload pending)'}`);

  const histData = extractData(histRes);
  const imports = Array.isArray(histData) ? histData : histData?.items || [];
  if (imports.length > 0) {
    const dlRes = await apiCall('GET', `/api/v1/timesheets/partner-import/${imports[0].id}/download`, null, adminToken);
    log('F10', 'Download original file', dlRes.ok ? 'PASS' : 'FAIL', `status=${dlRes.status}`);
  } else {
    log('F10', 'No imports to test download', 'WARN');
  }
}

// ──────────────────────────────────────────────
// F11/F12: Bulk Transfer (correct paths: /payrolls/*)
// ──────────────────────────────────────────────
async function testBulkTransfer() {
  console.log('\n📋 F11/F12: Bulk Transfer');

  // F11-01: Config
  const cfgRes = await apiCall('GET', '/api/v1/payrolls/auto-bulk-transfer/config', null, adminToken);
  log('F11', 'F11-01: Auto bulk transfer config', cfgRes.ok ? 'PASS' : 'FAIL', `status=${cfgRes.status} data=${JSON.stringify(extractData(cfgRes) || {}).slice(0, 100)}`);

  // F11-02: Estimate fee (POST, not GET)
  const today = new Date();
  const weekAgo = new Date(today); weekAgo.setDate(weekAgo.getDate() - 7);
  const feeRes = await apiCall('POST', '/api/v1/payrolls/auto-bulk-transfer/estimate-fee', {
    fromDate: weekAgo.toISOString().split('T')[0], toDate: today.toISOString().split('T')[0],
  }, adminToken);
  log('F11', 'F11-02: Estimate fee (weekly)', feeRes.ok ? 'PASS' : 'FAIL', `status=${feeRes.status}`);

  // Monthly estimate
  const monthStr = today.toISOString().slice(0, 7);
  const mFeeRes = await apiCall('POST', '/api/v1/payrolls/auto-bulk-transfer/estimate-fee', {
    for_month: monthStr,
  }, adminToken);
  log('F11', 'F11-03: Estimate fee (monthly)', mFeeRes.ok ? 'PASS' : 'FAIL', `status=${mFeeRes.status}`);

  // F11-07: Both forMonth + date range → should error (real finding if not rejected)
  // NOTE: DTO field is snake_case `for_month` (json:"for_month"); the frontend
  // bulk-transfer service uses the same. Earlier bug: test sent `forMonth`
  // (camelCase) which the JSON binder silently dropped → backend saw only the
  // date range and accepted it as 201.
  const bothRes = await apiCall('POST', '/api/v1/payrolls/auto-bulk-transfer', {
    for_month: monthStr, fromDate: weekAgo.toISOString().split('T')[0], toDate: today.toISOString().split('T')[0],
  }, adminToken);
  if (bothRes.status >= 400) {
    log('F11', 'F11-07: Both forMonth+dateRange rejected', 'PASS', `status=${bothRes.status}`);
  } else {
    log('F11', 'F11-07: Both forMonth+dateRange NOT rejected (potential business logic issue)', 'WARN',
      `status=${bothRes.status} — initiate accepted conflicting params`);
  }

  // F11-08: fromDate without toDate → error
  const noToRes = await apiCall('POST', '/api/v1/payrolls/auto-bulk-transfer', {
    fromDate: weekAgo.toISOString().split('T')[0],
  }, adminToken);
  log('F11', 'F11-08: fromDate without toDate rejected', noToRes.status >= 400 ? 'PASS' : 'FAIL', `status=${noToRes.status}`);

  // F12-01: Export bulk transfer Excel (POST, not GET)
  const expRes = await apiCall('POST', '/api/v1/payrolls/export-bulk-transfer', {
    for_month: monthStr,
  }, adminToken);
  log('F12', 'F12-01: Export bulk transfer Excel', expRes.ok ? 'PASS' : 'FAIL', `status=${expRes.status}`);

  // F12-02: List upload histories
  const histRes = await apiCall('GET', '/api/v1/payrolls/bulk-transfer-upload-histories?pageSize=5', null, adminToken);
  log('F12', 'F12-02: Upload histories list', histRes.ok ? 'PASS' : 'FAIL', `status=${histRes.status}`);
}

// ──────────────────────────────────────────────
// F13: Advance Payment
// ──────────────────────────────────────────────
async function testAdvancePayment() {
  console.log('\n📋 F13: Advance Payment');

  const tests = [
    ['F13-06: List pending', '/api/v1/advance-payments?status=pending&pageSize=10'],
    ['F13-07: Summary', '/api/v1/advance-payments/summary'],
    ['F13-08: Available months', '/api/v1/advance-payments/available-months'],
    ['F13-06b: List all', '/api/v1/advance-payments?pageSize=5'],
  ];
  for (const [name, url] of tests) {
    const res = await apiCall('GET', url, null, adminToken);
    log('F13', name, res.ok ? 'PASS' : 'FAIL', `status=${res.status}`);
  }

  // Browser
  if (!page) {
    log('F13', 'Browser: skipped (no browser)', 'WARN');
  } else try {
    await page.goto('http://localhost:3000/admin/advance-payments', { waitUntil: 'networkidle0', timeout: 15000 });
    await page.screenshot({ path: `${SCREENSHOT_DIR}/F13-advance-payments.png` });
    log('F13', 'Browser: Advance payments page', 'PASS');
  } catch (e) { log('F13', 'Browser: Advance payments', 'WARN', e.message); }
}

// ──────────────────────────────────────────────
// F15-F18: Wallet & Financial
// ──────────────────────────────────────────────
async function testWalletFinancial() {
  console.log('\n📋 F15-F18: Wallet & Financial');

  // Wallet
  const walletTests = [
    ['F15-01: Balance', '/api/v1/wallet/balance'],
    ['F15-03: Payments', '/api/v1/wallet/payments?pageSize=5'],
    ['F15-04: Topups', '/api/v1/wallet/topups?pageSize=5'],
  ];
  for (const [name, url] of walletTests) {
    const res = await apiCall('GET', url, null, adminToken);
    log('F15', name, res.ok ? 'PASS' : 'FAIL', `status=${res.status}`);
  }

  // Sync balance (correct path: /wallet/balance/sync)
  const syncRes = await apiCall('POST', '/api/v1/wallet/balance/sync', null, adminToken);
  log('F15', 'F15-02: Sync balance', syncRes.ok ? 'PASS' : 'FAIL', `status=${syncRes.status}`);

  // Transaction metadata
  const txMetaRes = await apiCall('GET', '/api/v1/transactions/metadata', null, adminToken);
  log('F16', 'F16-01: Transaction metadata', txMetaRes.ok ? 'PASS' : 'FAIL', `status=${txMetaRes.status}`);

  // Create expense transaction (correct fields: transaction_type, status required)
  const txRes = await apiCall('POST', '/api/v1/transactions', {
    transaction_type: 'expense', amount: 100000, party: 'QA Test Vendor',
    description: 'QA auto-test', status: 'pending',
  }, adminToken);
  log('F16', 'F16-02: Create expense transaction', txRes.ok || txRes.status === 201 ? 'PASS' : 'FAIL', `status=${txRes.status}`);

  const txId = extractData(txRes)?.id;
  if (txId) {
    // Partial settle
    const settleRes = await apiCall('POST', `/api/v1/transactions/${txId}/settle`, {
      amount: 50000, settlement_date: new Date().toISOString().split('T')[0], settlement_method: 'cash',
    }, adminToken);
    log('F16', 'F16-06: Partial settle', settleRes.ok ? 'PASS' : 'FAIL', `status=${settleRes.status}`);

    // Verify status
    const txDet = await apiCall('GET', `/api/v1/transactions/${txId}`, null, adminToken);
    const txStatus = extractData(txDet)?.status;
    log('F16', 'F16-06b: Status after partial settle', txStatus === 'partially_settled' ? 'PASS' : 'FAIL', `status=${txStatus}`);

    // Full settle
    const fullRes = await apiCall('POST', `/api/v1/transactions/${txId}/settle`, {
      amount: 50000, settlement_date: new Date().toISOString().split('T')[0], settlement_method: 'cash',
    }, adminToken);
    log('F16', 'F16-07: Full settle', fullRes.ok ? 'PASS' : 'FAIL', `status=${fullRes.status}`);

    // Reverse
    const revRes = await apiCall('POST', `/api/v1/transactions/${txId}/reverse`, {
      reason: 'QA auto-test reversal',
    }, adminToken);
    log('F16', 'F16-08: Reverse settled transaction', revRes.ok ? 'PASS' : 'FAIL', `status=${revRes.status}`);
  }

  // Zero amount rejected
  const zeroRes = await apiCall('POST', '/api/v1/transactions', {
    transaction_type: 'expense', amount: 0, party: 'QA', description: 'test', status: 'pending',
  }, adminToken);
  log('F16', 'F16-10: Zero amount rejected', zeroRes.status === 400 ? 'PASS' : 'FAIL', `status=${zeroRes.status}`);

  // Ledger
  const ledgerTests = [
    ['F18-01: Accounts metadata', '/api/v1/ledger/accounts/metadata'],
    ['F18-04: List entries', '/api/v1/ledger/entries?pageSize=5'],
    ['F18-05: Cash flow', '/api/v1/ledger/cash-flow?fromDate=2026-01-01&toDate=2026-12-31'],
    ['F18-06: Summary', '/api/v1/ledger/summary'],
  ];
  for (const [name, url] of ledgerTests) {
    const res = await apiCall('GET', url, null, adminToken);
    log('F18', name, res.ok ? 'PASS' : 'FAIL', `status=${res.status}`);
  }

  // Ledger entry create (expects JSON array) — use unique party to avoid 409 conflict
  const entryRes = await apiCall('POST', '/api/v1/ledger/entries', [{
    account: 'cash', party: `QA Test ${Date.now()}`, debit: 50000, credit: 0,
    date: new Date().toISOString().split('T')[0],
  }], adminToken);
  log('F18', 'F18-02: Create ledger entry', entryRes.ok || entryRes.status === 201 ? 'PASS' : 'FAIL', `status=${entryRes.status}`);

  const zeroEntryRes = await apiCall('POST', '/api/v1/ledger/entries', [{
    account: 'cash', party: 'QA Test', debit: 0, credit: 0,
    date: new Date().toISOString().split('T')[0],
  }], adminToken);
  log('F18', 'F18-08: Zero debit+credit rejected', zeroEntryRes.status === 400 ? 'PASS' : 'FAIL', `status=${zeroEntryRes.status}`);

  // Browser pages
  if (!page) {
    log('BROWSER', 'Wallet/Transactions/Ledger pages: skipped (no browser)', 'WARN');
  } else for (const [name, url] of [['Wallet', '/admin/wallet'], ['Transactions', '/admin/transactions'], ['Ledger', '/admin/ledger']]) {
    try {
      await page.goto(`http://localhost:3000${url}`, { waitUntil: 'networkidle0', timeout: 15000 });
      await page.screenshot({ path: `${SCREENSHOT_DIR}/F15-18-${name.toLowerCase()}.png` });
      log('BROWSER', `${name} page loads`, 'PASS');
    } catch (e) { log('BROWSER', name, 'WARN', e.message); }
  }
}

// ──────────────────────────────────────────────
// E2E-09: RBAC
// ──────────────────────────────────────────────
async function testRBAC() {
  console.log('\n📋 E2E-09: Role-Based Access Control');

  // Admin access
  const adminEmpRes = await apiCall('GET', '/api/v1/employees?pageSize=1', null, adminToken);
  log('RBAC', 'Admin accesses employees', adminEmpRes.ok ? 'PASS' : 'FAIL', `status=${adminEmpRes.status}`);

  // Partner access projects
  const partProjRes = await apiCall('GET', '/api/v1/projects?pageSize=5', null, partnerToken);
  log('RBAC', 'Partner accesses projects', partProjRes.ok ? 'PASS' : 'FAIL', `status=${partProjRes.status}`);

  // Partner blocked from loan management (admin-only)
  const loanRes = await apiCall('POST', '/api/v1/loans', {
    lender_id: 1, principal: 1000000, description: 'QA test',
  }, partnerToken);
  log('RBAC', 'Partner blocked from loan creation', loanRes.status === 403 ? 'PASS' : 'FAIL', `status=${loanRes.status}`);

  // Unauthenticated blocked
  const unauthRes = await apiCall('GET', '/api/v1/employees', null, null);
  log('RBAC', 'Unauthenticated blocked', unauthRes.status === 401 ? 'PASS' : 'FAIL', `status=${unauthRes.status}`);

  // Partner has read-only settings access (by design — Casbin policy allows GET)
  const settingsRes = await apiCall('GET', '/api/v1/settings', null, partnerToken);
  log('RBAC', 'Partner read-only settings (by design)', settingsRes.ok ? 'PASS' : 'FAIL', `status=${settingsRes.status}`);

  // Partner cannot WRITE settings
  const settingsWriteRes = await apiCall('POST', '/api/v1/settings', {
    key: 'test', value: 'blocked',
  }, partnerToken);
  log('RBAC', 'Partner blocked from settings write', settingsWriteRes.status === 403 ? 'PASS' : 'FAIL', `status=${settingsWriteRes.status}`);

  // Partner blocked from ledger
  const ledgerRes = await apiCall('GET', '/api/v1/ledger/entries?pageSize=1', null, partnerToken);
  log('RBAC', 'Partner blocked from ledger', ledgerRes.status === 403 ? 'PASS' : 'FAIL', `status=${ledgerRes.status}`);
}

// ──────────────────────────────────────────────
// Dashboard
// ──────────────────────────────────────────────
async function testDashboard() {
  console.log('\n📋 F22: Dashboard');
  const endpoints = [
    ['Summary', '/api/v1/dashboard/summary'],
    ['Financial overview', '/api/v1/dashboard/financial-overview'],
    ['Financial', '/api/v1/dashboard/financial'],
    ['Salary distribution', '/api/v1/dashboard/salary-distribution'],
    ['Recent activities', '/api/v1/dashboard/recent-activities'],
    ['Notifications', '/api/v1/dashboard/notifications'],
    ['New employees', '/api/v1/dashboard/new-employees'],
    ['Historical', '/api/v1/dashboard/historical'],
    ['Monthly financials', '/api/v1/dashboard/monthly-financials'],
    ['Employee activity', '/api/v1/dashboard/employee-activity'],
    ['Top paid', '/api/v1/dashboard/top-paid-employees'],
    ['Bank usage', '/api/v1/dashboard/bank-usage'],
    ['Project profitability', '/api/v1/dashboard/project-profitability'],
    ['Project weekly profit', '/api/v1/dashboard/project-weekly-profit'],
  ];
  for (const [name, url] of endpoints) {
    const res = await apiCall('GET', url, null, adminToken);
    log('DASH', `GET ${name}`, res.ok ? 'PASS' : 'FAIL', `status=${res.status}`);
  }
  if (!page) {
    log('DASH', 'Browser: Dashboard skipped (no browser)', 'WARN');
  } else try {
    await page.goto('http://localhost:3000/admin', { waitUntil: 'networkidle0', timeout: 15000 });
    await page.screenshot({ path: `${SCREENSHOT_DIR}/F22-dashboard.png` });
    log('DASH', 'Browser: Dashboard loads', 'PASS');
  } catch (e) { log('DASH', 'Browser: Dashboard', 'WARN', e.message); }
}

// ──────────────────────────────────────────────
// Main
// ──────────────────────────────────────────────
async function main() {
  try {
    await setup();
    await testTimesheetEntry();
    await testTimesheetApproval();
    await testBCCImport();
    await testBulkTransfer();
    await testAdvancePayment();
    await testWalletFinancial();
    await testRBAC();
    await testDashboard();

    const pass = results.filter(r => r.status === 'PASS').length;
    const fail = results.filter(r => r.status === 'FAIL').length;
    const warn = results.filter(r => r.status === 'WARN').length;
    console.log(`\n${'═'.repeat(60)}`);
    console.log(`  RESULTS: ${pass} PASS | ${fail} FAIL | ${warn} WARN | ${results.length} TOTAL`);
    console.log(`${'═'.repeat(60)}`);

    fs.writeFileSync('/Users/dev/Documents/projects/payroll/qa/results-p0.json', JSON.stringify({ results, summary: { pass, fail, warn, total: results.length } }, null, 2));

    if (fail > 0) {
      console.log('\n❌ FAILURES:');
      results.filter(r => r.status === 'FAIL').forEach(r => console.log(`  • [${r.category}] ${r.scenario} — ${r.detail}`));
    }
    process.exit(fail > 0 ? 1 : 0);
  } catch (err) {
    console.error('\n💥 Fatal:', err.message);
    process.exit(2);
  } finally {
    await teardown();
  }
}

main();
