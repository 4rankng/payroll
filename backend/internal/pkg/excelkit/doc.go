// Package excelkit holds the generic Excel-parsing primitives shared by all
// workbook-import islands (BCC timesheets, OnePay fee reports, employee
// imports, wallet bulk, settlement sao-kê).
//
// Rules:
//   - Zero business logic. Domain-specific fingerprints ("Tổng cộng" stop
//     rows, shift labels, day-column scans) stay in their owning services.
//   - Every function here is a verbatim port of an existing, battle-tested
//     implementation; semantics are frozen and locked by table tests. Do not
//     "improve" matching behavior here — that would change which production
//     files parse, and the per-parser goldens would rightly fail.
//   - Divergent normalizers are kept divergent and distinctly named. The
//     space-stripping vs space-keeping Vietnamese name normalizers are a
//     bank-reconciliation concern and are deliberately NOT unified here.
package excelkit
