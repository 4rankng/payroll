package converter

import (
	"api-server/internal/app/dto"
	"api-server/internal/domain"
)

// MapNewEmployeeItems converts a batch of domain employees into DTOs.
func MapNewEmployeeItems(employees []*domain.Employee, projects map[uint]*domain.Project) []dto.NewEmployeeItem {
	employeeItems := make([]dto.NewEmployeeItem, 0, len(employees))

	for _, emp := range employees {
		var dateOfBirth string
		if emp.DateOfBirth != nil && !emp.DateOfBirth.IsZero() {
			dateOfBirth = emp.DateOfBirth.Format("2006-01-02")
		}

		createdByName := "System"
		if emp.Creator.Fullname != "" {
			createdByName = emp.Creator.Fullname
		} else if emp.Creator.Username != "" {
			createdByName = emp.Creator.Username
		}

		item := dto.NewEmployeeItem{
			ID:             int(emp.ID),
			Fullname:       emp.Fullname,
			Email:          emp.Email,
			CCCD:           emp.CCCD,
			DateOfBirth:    dateOfBirth,
			CreatedAt:      emp.CreatedAt,
			CreatedByName:  createdByName,
			Avatar:         nil,
			CurrentProject: nil,
		}

		if projects != nil {
			if project, ok := projects[emp.ID]; ok && project != nil {
				item.CurrentProject = &dto.ProjectInfo{
					ID:   int(project.ID),
					Name: project.Name,
					Code: project.Code,
				}
			}
		}

		employeeItems = append(employeeItems, item)
	}

	return employeeItems
}
