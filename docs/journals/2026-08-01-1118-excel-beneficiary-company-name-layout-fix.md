# Excel Beneficiary Company-Name Layout Fix

**Date**: 2026-08-01 11:18 +08
**Severity**: Medium
**Component**: Payroll Excel templates
**Status**: Resolved

## What Happened

After the beneficiary bank details were updated to `CONG TY TNHH MTV GPPM TING TING / 283866888`, the company name no longer fit the existing Excel layout in `backend/templates/payroll_template.xlsx` and `backend/templates/sao_ke_tt_theo_du_an.xlsx`. The first response was a blunt column-wide fix in commit `61c93dc`, which widened column E everywhere to 38 so the holder name would not clip.

That solved the clipping, but it also changed the print geometry of both templates. These workbooks have `fitToPage="true"`, so widening a core data column is not a free change. It degrades the sheet’s scale and makes the exported report feel visibly off.

The banner slot in `backend/templates/payroll_template.xlsx` also needed a real fix. The obsolete black `P / HỆ THỐNG ỨNG LƯƠNG PAYROLL SYSTEM` artwork was removed, `B8` was left blank, and the workbook now embeds the exact `backend/templates/email/banner.jpg` bytes in the `Payroll Report!A8` one-cell anchor. That source image is 1280x512, so it renders at 320x128 with the intended 2.5:1 ratio instead of a stretched placeholder.

## The Brutal Truth

The first fix was technically correct and operationally wrong. It traded one visible defect for a quieter one, which is exactly the kind of spreadsheet bug that gets missed until somebody prints the file and wonders why the layout suddenly feels bloated. That is annoying, because the real problem was not the company name itself. It was our refusal to preserve the original page fit and solve the overflow at the row level.

## Technical Details

- `58df1ef` updated the static account holder and account number in the email and workbook templates.
- `61c93dc` widened column E globally to 38 in both Excel templates.
- The final workbook state restores the original E widths: `21.140625` in `payroll_template.xlsx` and `24.28515625` in `sao_ke_tt_theo_du_an.xlsx`.
- Only the holder rows were expanded to `32` points and set to `wrapText` plus `vertical="center"` so the two-line company name stays readable without changing the rest of the sheet.
- `payroll_template.xlsx` keeps `B8` empty, anchors the banner at `A8` as a `oneCellAnchor` sized to `320x128`, and the embedded `xl/media/image2.jpeg` bytes match `backend/templates/email/banner.jpg` exactly.
- The existing logo remains in the workbook as a separate image; only the banner slot changed.
- A focused regression test was added in [`backend/internal/app/services/payroll/report_template_test.go`](/Users/dev/Documents/projects/payroll/backend/internal/app/services/payroll/report_template_test.go) to assert the beneficiary value, column width, merged range, row height, alignment, placeholder cell, and banner byte parity.

## What We Tried

1. Widened column E across both templates in `61c93dc`.
2. Replaced that with a narrower layout that keeps the existing print scale and only increases the beneficiary holder rows.
3. Verified the workbook contract with `go test ./internal/app/services/payroll -run 'TestPayrollStatementTemplatesKeepCompanyBeneficiaryReadable|TestPayrollTemplateUsesMergedBeneficiaryValueRange|TestProjectStatementTemplateGivesWrappedBeneficiaryEnoughHeight|TestPayrollTemplateUsesTingTingBanner' -count=1`, which passed.
4. Rendered `backend/templates/payroll_template.xlsx` through LibreOffice headlessly to PDF and confirmed the workbook exports cleanly with the banner image intact.

## Root Cause Analysis

The bug came from treating a content-length problem like a column-width problem. The workbook needed room for a longer beneficiary name, but the global column resize changed the page layout for every row in the sheet. The correct constraint was not “make column E bigger everywhere”; it was “keep the sheet width stable and give the holder row enough vertical space to wrap.”

## Lessons Learned

- Do not widen a print-sensitive workbook column globally unless the page geometry has been rechecked.
- When a spreadsheet cell overflows because a label got longer, prefer row height and wrapping before changing the column scale.
- When a workbook already has a branded banner slot, keep the placeholder cell blank and swap the exact asset bytes instead of redrawing the artwork.
- Add a regression test when the fix depends on workbook metadata, not just visible cell values.

## Next Steps

The integration target was attempted from `backend/`, but the local API was not running on port 8080, so it stopped at discovery before changing test data. The full Go suite was also run: the affected payroll and FlexPay packages passed, while unrelated existing configuration and 9Pay tests remained red. The focused workbook test covers the changed contract; if the templates or beneficiary text change again, this same test should fail before anyone ships another bad width fix.
