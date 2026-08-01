package payroll

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

var (
	templateRootElementPattern = regexp.MustCompile(`<([A-Za-z_][\w.-]*:)?[A-Za-z_][\w.-]*\b[^>]*>`)
	templateMCIgnorablePattern = regexp.MustCompile(`\s+mc:Ignorable="([^"]*)"`)
)

func TestPayrollStatementTemplatesDeclareCompatibilityPrefixes(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"payroll_template.xlsx", "sao_ke_tt_theo_du_an.xlsx"} {
		filename := filename
		t.Run(filename, func(t *testing.T) {
			t.Parallel()

			workbook, err := zip.OpenReader(templatePath(t, filename))
			if err != nil {
				t.Fatalf("open template archive: %v", err)
			}
			t.Cleanup(func() {
				if err := workbook.Close(); err != nil {
					t.Errorf("close template archive: %v", err)
				}
			})

			for _, part := range workbook.File {
				if !strings.HasSuffix(part.Name, ".xml") {
					continue
				}
				content := readTemplatePart(t, part)
				rootElement := templateRootElementPattern.Find(content)
				for _, match := range templateMCIgnorablePattern.FindAllSubmatch(rootElement, -1) {
					for _, prefix := range strings.Fields(string(match[1])) {
						if !bytes.Contains(rootElement, []byte("xmlns:"+prefix+"=")) {
							t.Fatalf("%s has undeclared mc:Ignorable prefix %q", part.Name, prefix)
						}
					}
				}
			}
		})
	}
}

func readTemplatePart(t *testing.T, part *zip.File) []byte {
	t.Helper()

	reader, err := part.Open()
	if err != nil {
		t.Fatalf("open %s: %v", part.Name, err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close %s: %v", part.Name, err)
		}
	}()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read %s: %v", part.Name, err)
	}
	return content
}

func TestPayrollStatementTemplatesKeepCompanyBeneficiaryReadable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		sheet       string
		holderCell  string
		accountCell string
		bankCell    string
		columnWidth float64
	}{
		{
			name:        "payroll report",
			filename:    "payroll_template.xlsx",
			sheet:       "Payroll Report",
			holderCell:  "E9",
			accountCell: "E10",
			bankCell:    "E11",
			columnWidth: 21.140625,
		},
		{
			name:        "project statement",
			filename:    "sao_ke_tt_theo_du_an.xlsx",
			sheet:       "Summary",
			holderCell:  "E8",
			accountCell: "E9",
			bankCell:    "E10",
			columnWidth: 24.28515625,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			workbook, err := excelize.OpenFile(templatePath(t, tt.filename))
			if err != nil {
				t.Fatalf("open template: %v", err)
			}
			t.Cleanup(func() {
				if err := workbook.Close(); err != nil {
					t.Errorf("close template: %v", err)
				}
			})

			assertTemplateCell(t, workbook, tt.sheet, tt.holderCell, "CONG TY TNHH MTV GPPM TING TING")
			assertTemplateCell(t, workbook, tt.sheet, tt.accountCell, "283866888")
			assertTemplateCell(t, workbook, tt.sheet, tt.bankCell, "TECHCOMBANK")

			width, err := workbook.GetColWidth(tt.sheet, "E")
			if err != nil {
				t.Fatalf("read beneficiary column width: %v", err)
			}
			if math.Abs(width-tt.columnWidth) > 0.01 {
				t.Fatalf("beneficiary column E width = %.2f, want %.2f to preserve the report's print scale", width, tt.columnWidth)
			}
		})
	}
}

func TestPayrollTemplateUsesMergedBeneficiaryValueRange(t *testing.T) {
	t.Parallel()

	workbook, err := excelize.OpenFile(templatePath(t, "payroll_template.xlsx"))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	t.Cleanup(func() {
		if err := workbook.Close(); err != nil {
			t.Errorf("close template: %v", err)
		}
	})

	mergedCells, err := workbook.GetMergeCells("Payroll Report")
	if err != nil {
		t.Fatalf("read merged cells: %v", err)
	}
	for _, mergedCell := range mergedCells {
		if mergedCell.GetStartAxis() == "E9" && mergedCell.GetEndAxis() == "F9" {
			assertWrappedBeneficiaryRow(t, workbook, "Payroll Report", "E9", 9)
			return
		}
	}
	t.Fatal("beneficiary holder value must remain merged across Payroll Report!E9:F9")
}

func TestProjectStatementTemplateGivesWrappedBeneficiaryEnoughHeight(t *testing.T) {
	t.Parallel()

	workbook, err := excelize.OpenFile(templatePath(t, "sao_ke_tt_theo_du_an.xlsx"))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	t.Cleanup(func() {
		if err := workbook.Close(); err != nil {
			t.Errorf("close template: %v", err)
		}
	})

	assertWrappedBeneficiaryRow(t, workbook, "Summary", "E8", 8)
}

func TestPayrollTemplateUsesTingTingBanner(t *testing.T) {
	t.Parallel()

	workbook, err := excelize.OpenFile(templatePath(t, "payroll_template.xlsx"))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	t.Cleanup(func() {
		if err := workbook.Close(); err != nil {
			t.Errorf("close template: %v", err)
		}
	})

	placeholderText, err := workbook.GetCellValue("Payroll Report", "B8")
	if err != nil {
		t.Fatalf("read placeholder text: %v", err)
	}
	if placeholderText != "" {
		t.Fatalf("Payroll Report!B8 = %q, want the obsolete Payroll System text removed", placeholderText)
	}

	pictures, err := workbook.GetPictures("Payroll Report", "A8")
	if err != nil {
		t.Fatalf("read payroll banner: %v", err)
	}
	if len(pictures) != 1 {
		t.Fatalf("pictures anchored at Payroll Report!A8 = %d, want exactly one TingTing banner", len(pictures))
	}
	wantBanner, err := os.ReadFile(templatePath(t, filepath.Join("email", "banner.jpg")))
	if err != nil {
		t.Fatalf("read source banner: %v", err)
	}
	if !bytes.Equal(pictures[0].File, wantBanner) {
		t.Fatal("Payroll Report!A8 does not embed templates/email/banner.jpg")
	}
	bannerFormat := pictures[0].Format
	if bannerFormat == nil {
		t.Fatal("Payroll Report!A8 banner has no drawing format")
	}
	if math.Abs(bannerFormat.ScaleX-0.25) > 0.001 || math.Abs(bannerFormat.ScaleY-0.25) > 0.001 {
		t.Fatalf("banner scale = %.3f x %.3f, want 0.25 x 0.25 (320 x 128 px)", bannerFormat.ScaleX, bannerFormat.ScaleY)
	}
	if bannerFormat.OffsetX != 0 || bannerFormat.OffsetY != 0 || bannerFormat.Positioning != "oneCell" {
		t.Fatalf("banner anchor = offset (%d, %d), positioning %q; want A8 with zero offset and oneCell positioning", bannerFormat.OffsetX, bannerFormat.OffsetY, bannerFormat.Positioning)
	}
	if !bannerFormat.LockAspectRatio || bannerFormat.PrintObject == nil || !*bannerFormat.PrintObject {
		t.Fatalf("banner drawing options = %#v, want locked 2.5:1 aspect ratio and printing enabled", bannerFormat)
	}

	companyLogos, err := workbook.GetPictures("Payroll Report", "A1")
	if err != nil {
		t.Fatalf("read company logo: %v", err)
	}
	if len(companyLogos) != 1 {
		t.Fatalf("pictures anchored at Payroll Report!A1 = %d, want the existing company logo preserved", len(companyLogos))
	}
	const companyLogoSHA256 = "ad1c9591d7b8edcf9b0b8b9fecdb82f87e00101d404ec24862f6e9cdef72804f"
	if got := sha256.Sum256(companyLogos[0].File); fmt.Sprintf("%x", got) != companyLogoSHA256 {
		t.Fatalf("Payroll Report!A1 company logo changed: sha256 = %x, want %s", got, companyLogoSHA256)
	}

	pageLayout, err := workbook.GetPageLayout("Payroll Report")
	if err != nil {
		t.Fatalf("read payroll page layout: %v", err)
	}
	if pageLayout.Size == nil || *pageLayout.Size != 9 || pageLayout.Orientation == nil || *pageLayout.Orientation != "portrait" {
		t.Fatalf("payroll page layout = %#v, want A4 portrait", pageLayout)
	}
}

func assertWrappedBeneficiaryRow(t *testing.T, workbook *excelize.File, sheet, cell string, row int) {
	t.Helper()

	rowHeight, err := workbook.GetRowHeight(sheet, row)
	if err != nil {
		t.Fatalf("read beneficiary row height: %v", err)
	}
	if rowHeight < 32 {
		t.Fatalf("beneficiary row height = %.2f, want at least 32 for the two-line company name", rowHeight)
	}

	styleID, err := workbook.GetCellStyle(sheet, cell)
	if err != nil {
		t.Fatalf("read beneficiary style: %v", err)
	}
	style, err := workbook.GetStyle(styleID)
	if err != nil {
		t.Fatalf("resolve beneficiary style: %v", err)
	}
	if style.Alignment == nil || !style.Alignment.WrapText || style.Alignment.Vertical != "center" {
		t.Fatalf("beneficiary alignment = %#v, want wrapped and vertically centered", style.Alignment)
	}
}

func templatePath(t *testing.T, filename string) string {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}

	return filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "..", "templates", filename)
}

func assertTemplateCell(t *testing.T, workbook *excelize.File, sheet, cell, want string) {
	t.Helper()

	got, err := workbook.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("read %s!%s: %v", sheet, cell, err)
	}
	if got != want {
		t.Fatalf("%s!%s = %q, want %q", sheet, cell, got, want)
	}
}
