package flex_pay

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	appconfig "api-server/internal/app/services/config"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

var testMCIgnorablePattern = regexp.MustCompile(`\s+mc:Ignorable="([^"]*)"`)

func TestGenerateExcelDoesNotCreateOrphanedTableRelationships(t *testing.T) {
	chdirBackendRoot(t)

	paidAt := time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)
	reportData := []*domainServices.ProjectFlexPayReportData{
		{
			Project: &domain.Project{
				ID:   1,
				Name: "LGD",
			},
			SalaryPeriodFrom:     time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
			SalaryPeriodTo:       time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC),
			TotalAmount:          980000,
			TotalRequestedAmount: 1000000,
			EmployeeCount:        1,
			RequestIDs:           []uint{1001},
			EmployeeData: []*domainServices.EmployeeFlexPayReportData{
				{
					EmployeeName:   "Nguyen Van A",
					EmployeeCCCD:   "012345678901",
					ProjectName:    "LGD",
					PaymentDate:    &paidAt,
					TotalPaid:      980000,
					TotalRequested: 1000000,
					RequestIDs:     []uint{1001},
				},
			},
		},
	}

	bytes, _, err := NewFlexPayReconciliationExporter(nil).GenerateExcel(context.Background(), reportData, time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, assertNoOrphanedWorksheetTableRelationships(bytes))
	require.NoError(t, assertContiguousWorksheetParts(bytes))
	require.NoError(t, assertNoUndeclaredIgnorablePrefixes(bytes))
}

func assertNoOrphanedWorksheetTableRelationships(workbook []byte) error {
	files, err := unzipWorkbookFiles(workbook)
	if err != nil {
		return err
	}

	for relsPath, relsContent := range files {
		if !strings.HasPrefix(relsPath, "xl/worksheets/_rels/sheet") || !strings.HasSuffix(relsPath, ".xml.rels") {
			continue
		}

		var rels workbookRelationships
		if err := xml.Unmarshal(relsContent, &rels); err != nil {
			return fmt.Errorf("parse %s: %w", relsPath, err)
		}

		sheetPath := strings.TrimSuffix(strings.Replace(relsPath, "xl/worksheets/_rels/", "xl/worksheets/", 1), ".rels")
		sheetContent, ok := files[sheetPath]
		if !ok {
			return fmt.Errorf("missing worksheet XML for %s", relsPath)
		}

		for _, rel := range rels.Relationships {
			if !strings.HasSuffix(rel.Type, "/table") {
				continue
			}
			if !bytes.Contains(sheetContent, []byte("<tableParts")) {
				return fmt.Errorf("%s has table relationship %s without tableParts in %s", relsPath, rel.ID, sheetPath)
			}
			targetPath := path.Clean(path.Join("xl/worksheets", rel.Target))
			if _, ok := files[targetPath]; !ok {
				return fmt.Errorf("%s references missing table target %s", relsPath, targetPath)
			}
		}
	}

	return nil
}

func unzipWorkbookFiles(workbook []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(workbook), int64(len(workbook)))
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", file.Name, err)
		}
		content, readErr := io.ReadAll(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s: %w", file.Name, closeErr)
		}
		files[file.Name] = content
	}

	return files, nil
}

type workbookRelationships struct {
	Relationships []workbookRelationship `xml:"Relationship"`
}

type workbookRelationship struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

func assertContiguousWorksheetParts(workbook []byte) error {
	files, err := unzipWorkbookFiles(workbook)
	if err != nil {
		return err
	}

	for _, worksheetPath := range []string{
		"xl/worksheets/sheet1.xml",
		"xl/worksheets/sheet2.xml",
		"xl/worksheets/sheet3.xml",
	} {
		if _, ok := files[worksheetPath]; !ok {
			return fmt.Errorf("missing worksheet part %s", worksheetPath)
		}
	}
	if _, ok := files["xl/worksheets/sheet4.xml"]; ok {
		return fmt.Errorf("unexpected worksheet gap: found sheet4.xml for three-sheet workbook")
	}

	return nil
}

func assertNoUndeclaredIgnorablePrefixes(workbook []byte) error {
	files, err := unzipWorkbookFiles(workbook)
	if err != nil {
		return err
	}

	for fileName, content := range files {
		if !strings.HasSuffix(fileName, ".xml") || !bytes.Contains(content, []byte("mc:Ignorable=")) {
			continue
		}

		xmlContent := string(content)
		matches := testMCIgnorablePattern.FindAllStringSubmatch(xmlContent, -1)
		for _, match := range matches {
			if len(match) != 2 {
				continue
			}
			for _, prefix := range strings.Fields(match[1]) {
				if !strings.Contains(xmlContent, "xmlns:"+prefix+"=") {
					return fmt.Errorf("%s has undeclared mc:Ignorable prefix %s", fileName, prefix)
				}
			}
		}
	}

	return nil
}

func chdirBackendRoot(t *testing.T) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime caller must be available")

	backendRoot := path.Clean(path.Join(path.Dir(filename), "../../../.."))
	previousWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(backendRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previousWD))
	})
}

// stubFlexPayBankProvider feeds the exporter a FlexPay-only beneficiary so the
// test proves the FlexPay statement reads its own account, not the weekly one.
type stubFlexPayBankProvider struct {
	info appconfig.TransferBankInfo
}

func (s stubFlexPayBankProvider) GetFlexPayTransferBankInfo(context.Context) appconfig.TransferBankInfo {
	return s.info
}

func TestFlexPayExcelShowsConfiguredBank(t *testing.T) {
	chdirBackendRoot(t)

	reportData := []*domainServices.ProjectFlexPayReportData{
		{Project: &domain.Project{ID: 1, Name: "Dự án A"}, TotalAmount: 500_000, RequestIDs: []uint{1}},
	}
	provider := stubFlexPayBankProvider{info: appconfig.TransferBankInfo{
		Holder: "CONG TY FLEXPAY",
		Number: "444555666",
		Name:   "Ngân hàng FlexPay",
	}}
	out, _, err := NewFlexPayReconciliationExporter(provider).GenerateExcel(
		context.Background(), reportData, time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(out))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	for cell, want := range map[string]string{
		"E8":  "CONG TY FLEXPAY",
		"E9":  "444555666",
		"E10": "Ngân hàng FlexPay",
	} {
		got, err := f.GetCellValue("Summary", cell)
		require.NoErrorf(t, err, "read %s", cell)
		require.Equalf(t, want, got, "Summary!%s", cell)
	}
}

// TestFlexPayEmailUsesFlexPayAccount exercises the same composition the manual
// reconciliation handler uses: the exporter's BankInfoForStatement feeds
// BuildSaoKeEmailBodies. The statement email must print the FlexPay account,
// not the weekly payroll one.
func TestFlexPayEmailUsesFlexPayAccount(t *testing.T) {
	provider := stubFlexPayBankProvider{info: appconfig.TransferBankInfo{
		Holder: "CONG TY FLEXPAY",
		Number: "444555666",
		Name:   "Ngân hàng FlexPay",
	}}
	exporter := NewFlexPayReconciliationExporter(provider)

	htmlBody, textBody := BuildSaoKeEmailBodies(
		"2026-06", "31/07/2026", "118.110.000 đ", exporter.BankInfoForStatement(context.Background()),
	)

	for _, body := range []string{htmlBody, textBody} {
		for _, want := range []string{"CONG TY FLEXPAY", "444555666", "Ngân hàng FlexPay"} {
			require.Containsf(t, body, want, "FlexPay statement email must print the FlexPay account")
		}
		require.NotContains(t, body, appconfig.DefaultTransferBankHolder,
			"FlexPay statement email must not fall back to the weekly account")
	}
}
