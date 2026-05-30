package main

import (
	"fmt"
)

const flowPartner = "PartnerTimesheet"

func runPartnerTimesheetTests(client *APIClient, data *TestData, reporter *Reporter) {
	reporter.PrintSection("FLOW 4: Partner Creates Timesheets → Admin Approves")

	if len(data.Partners) == 0 {
		reporter.Skip(flowPartner, "All partner timesheet tests", "no partner users found")
		return
	}

	if data.WeeklyProject == nil {
		reporter.Skip(flowPartner, "All partner timesheet tests", "no weekly project found")
		return
	}

	_ = fmt.Sprintf
}
