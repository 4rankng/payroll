// bccinspect is a diagnostic CLI for BCC workbook parsing: it runs the same
// routing the import service uses (registry detection + strategy fallback)
// and prints what parsed. Previously it called only the legacy parser, which
// misreported date-row files — it now tells the truth.
package main

import (
	"fmt"
	"os"

	excelparser "api-server/internal/app/services/excel"

	"github.com/xuri/excelize/v2"
)

func main() {
	for _, p := range os.Args[1:] {
		f, err := excelize.OpenFile(p)
		if err != nil {
			fmt.Println(err)
			continue
		}

		det, derr := excelparser.DetectFormat(f)
		if derr != nil {
			fmt.Printf("%-50s DETECT ERR: %v\n", p, derr)
			_ = f.Close()
			continue
		}
		fmt.Printf("%-50s format=%s\n", p, formatName(det.Format))

		switch det.Format {
		case excelparser.FormatLegacy, excelparser.FormatDateRow:
			// Both flow through the strategy router in the import service —
			// a "BCC"-named sheet can carry a date-row layout (T09).
			data, winner, perr := excelparser.ParseBCCData(f)
			if perr != nil {
				fmt.Printf("    ERR: %v\n", perr)
				break
			}
			fmt.Printf("    OK: winner=%s, %d employees, %d shift rates\n",
				formatName(winner), len(data.Employees), len(data.ShiftRates))
			dumpFirstEmployee(data.Employees)
		case excelparser.FormatWeeklyBCC:
			data, perr := excelparser.ParseWeeklyBCCFile(f, det.WeeklyBCCSheets)
			if perr != nil {
				fmt.Printf("    ERR: %v\n", perr)
				break
			}
			for _, s := range data.Sheets {
				fmt.Printf("    OK: sheet BCC-%s — %d employees\n", s.ShiftType, len(s.Employees))
			}
		case excelparser.FormatWeeklyPayment:
			data, perr := excelparser.ParseWeeklyPaymentFile(f, det.WeeklyPaymentSheets, os.Getenv("FOR_MONTH"))
			if perr != nil {
				fmt.Printf("    ERR: %v\n", perr)
				break
			}
			for _, s := range data.Sheets {
				fmt.Printf("    OK: sheet %q — %d employees\n", s.Position, len(s.Employees))
			}
		case excelparser.FormatMultiPosition:
			data, perr := excelparser.ParseMultiPositionFile(f, det.PositionSheets)
			if perr != nil {
				fmt.Printf("    ERR: %v\n", perr)
				break
			}
			for _, s := range data.Sheets {
				fmt.Printf("    OK: sheet %q — %d employees\n", s.Position, len(s.Employees))
			}
		}

		_ = f.Close()
	}
}

func dumpFirstEmployee(employees []excelparser.BCCEmployeeData) {
	if len(employees) == 0 {
		return
	}
	e := employees[0]
	fmt.Printf("    1st emp: %q (%s) — %d entries; sample: ", e.FullName, e.EmployeeCode, len(e.Entries))
	for i := 0; i < 3 && i < len(e.Entries); i++ {
		fmt.Printf("d%d=%s:%.1fh ", e.Entries[i].DayNum, e.Entries[i].ShiftLabel, e.Entries[i].Hours)
	}
	fmt.Println()
}

func formatName(format excelparser.BCCFormat) string {
	for _, entry := range []struct {
		f    excelparser.BCCFormat
		name string
	}{
		{excelparser.FormatLegacy, "legacy"},
		{excelparser.FormatMultiPosition, "multi-position"},
		{excelparser.FormatWeeklyBCC, "weekly-bcc"},
		{excelparser.FormatWeeklyPayment, "weekly-payment"},
		{excelparser.FormatDateRow, "date-row"},
	} {
		if entry.f == format {
			return entry.name
		}
	}
	return fmt.Sprintf("unknown(%d)", int(format))
}
