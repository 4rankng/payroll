package excel

import (
	"fmt"
	"sort"
	"strings"

	"api-server/internal/domain"
)

// offenderListLimit caps how many employees the refusal names before
// summarising the rest, so a large batch still yields a readable message.
const offenderListLimit = 5

// ValidateOneProjectPerCycle refuses a transfer batch in which one employee is
// paid for more than one project in the same cycle.
//
// The batch is keyed by (employee, project) — planner.aggregateTimesheetData
// mints one transaction code per key — so an employee with approved timesheets
// in two projects in one cycle produces two rows carrying the same bank
// account. The bank pays both: the employee receives their salary as two
// credits and reconciliation cannot tell which of the two was filed by
// mistake. Two projects in one cycle is always a data-entry error (a partner
// uploaded timesheets under the wrong project); the rows must be deleted or
// moved before money moves.
//
// Fatal rather than a skipped row on purpose: dropping one of the two projects
// silently would pay an incomplete salary.
func ValidateOneProjectPerCycle(data *BulkTransferData) error {
	if data == nil {
		return nil
	}

	projectsByEmployee := make(map[uint]map[uint]struct{}, len(data.EmployeeProjectAmounts))
	for key := range data.EmployeeProjectAmounts {
		projects, ok := projectsByEmployee[key.EmployeeID]
		if !ok {
			projects = make(map[uint]struct{}, 1)
			projectsByEmployee[key.EmployeeID] = projects
		}
		projects[key.ProjectID] = struct{}{}
	}

	offenders := make([]uint, 0)
	for employeeID, projects := range projectsByEmployee {
		if len(projects) > 1 {
			offenders = append(offenders, employeeID)
		}
	}
	if len(offenders) == 0 {
		return nil
	}
	sort.Slice(offenders, func(i, j int) bool { return offenders[i] < offenders[j] })

	return domain.NewValidationErrorWithCode(
		domain.ErrCodeMultipleProjectsInCycle,
		multipleProjectsMessage(data, offenders, projectsByEmployee),
	)
}

func multipleProjectsMessage(data *BulkTransferData, offenders []uint, projectsByEmployee map[uint]map[uint]struct{}) string {
	listed := offenders
	if len(listed) > offenderListLimit {
		listed = listed[:offenderListLimit]
	}

	details := make([]string, 0, len(listed))
	for _, employeeID := range listed {
		details = append(details, fmt.Sprintf("%s (%s)", employeeLabel(data, employeeID), projectLabels(data, projectsByEmployee[employeeID])))
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Không thể xuất chuyển lô: %d nhân viên có chấm công ở nhiều dự án trong cùng kỳ lương — %s",
		len(offenders), strings.Join(details, "; "))
	if remaining := len(offenders) - len(listed); remaining > 0 {
		fmt.Fprintf(&builder, "; và %d nhân viên khác", remaining)
	}
	builder.WriteString(". Mỗi nhân viên chỉ được chấm công ở một dự án trong một kỳ lương — " +
		"xóa hoặc chuyển các dòng chấm công sai dự án rồi xuất lại.")
	return builder.String()
}

func employeeLabel(data *BulkTransferData, employeeID uint) string {
	if name := strings.TrimSpace(data.EmployeeData[employeeID].Fullname); name != "" {
		return name
	}
	return fmt.Sprintf("Nhân viên #%d", employeeID)
}

func projectLabels(data *BulkTransferData, projectIDs map[uint]struct{}) string {
	ids := make([]uint, 0, len(projectIDs))
	for projectID := range projectIDs {
		ids = append(ids, projectID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	labels := make([]string, 0, len(ids))
	for _, projectID := range ids {
		if name := strings.TrimSpace(data.ProjectData[projectID].Name); name != "" {
			labels = append(labels, name)
			continue
		}
		labels = append(labels, fmt.Sprintf("Dự án #%d", projectID))
	}
	return strings.Join(labels, ", ")
}
