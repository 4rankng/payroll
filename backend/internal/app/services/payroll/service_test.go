package payroll

import (
	"context"
	"testing"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPayrollService_GetPayrollHistories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockTimesheetRepo := mocks.NewMockTimesheetRepository(ctrl)

	service := &PayrollService{
		timesheetRepo: mockTimesheetRepo,
	}

	// Setup test data
	fromDate := "2023-01-01"
	toDate := "2023-01-31"

	request := &dto.ListPayrollHistoriesRequest{
		FromDate:  fromDate,
		ToDate:    toDate,
		Page:      1,
		PageSize:  10,
		SortBy:    "paid_date",
		SortOrder: "desc",
	}

	userID := uint(1)
	userRole := "admin" // Admin should see all records

	expectedHistories := []*domain.PaymentHistory{
		{
			EmployeeID:      1,
			EmployeeName:    "John Doe",
			EmployeeCCCD:    "123456789",
			ProjectID:       1,
			ProjectName:     "Project Alpha",
			Position:        "Developer",
			TotalPaidAmount: 10000000,
			PaidAt:          time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			EmployeeID:      2,
			EmployeeName:    "Jane Smith",
			EmployeeCCCD:    "987654321",
			ProjectID:       2,
			ProjectName:     "Project Beta",
			Position:        "Designer",
			TotalPaidAmount: 8000000,
			PaidAt:          time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC),
		},
	}

	totalCount := int64(2)

	loc, _ := time.LoadLocation("Local")
	fromDateParsed, _ := time.ParseInLocation("2006-01-02", fromDate, loc)
	toDateParsed, _ := time.ParseInLocation("2006-01-02", toDate, loc)

	filters := domain.PaymentHistoryFilters{
		FromDate:  &fromDateParsed,
		ToDate:    &toDateParsed,
		SortBy:    "paid_date",
		SortOrder: "desc",
		Limit:     10,
		Offset:    0,
	}

	mockTimesheetRepo.EXPECT().GetPaymentHistories(gomock.Any(), filters).Return(expectedHistories, totalCount, nil)

	response, err := service.GetPayrollHistories(context.Background(), request, userID, userRole)

	assert.NoError(t, err)
	assert.Equal(t, len(expectedHistories), len(response.Data))
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 10, response.Pagination.PageSize)
	assert.Equal(t, 1, response.Pagination.TotalPages)
	assert.Equal(t, int64(2), response.Pagination.TotalRecords)

	// Check that the first item is properly mapped
	assert.Equal(t, uint(1), response.Data[0].EmployeeID)
	assert.Equal(t, "John Doe", response.Data[0].EmployeeName)
	assert.Equal(t, "123456789", response.Data[0].EmployeeCCCD)
	assert.Equal(t, uint(1), response.Data[0].ProjectID)
	assert.Equal(t, "Project Alpha", response.Data[0].ProjectName)
	assert.Equal(t, "Developer", response.Data[0].Position)
	assert.Equal(t, int64(10000000), response.Data[0].TotalPaidAmount)
}

func TestPayrollService_GetPayrollHistories_WithPartnerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockTimesheetRepo := mocks.NewMockTimesheetRepository(ctrl)

	service := &PayrollService{
		timesheetRepo: mockTimesheetRepo,
	}

	// Setup test data
	fromDate := "2023-01-01"
	toDate := "2023-01-31"

	request := &dto.ListPayrollHistoriesRequest{
		FromDate:  fromDate,
		ToDate:    toDate,
		Page:      1,
		PageSize:  10,
		SortBy:    "paid_date",
		SortOrder: "desc",
	}

	userID := uint(1)
	userRole := "partner" // Partner should only see their own records

	loc, _ := time.LoadLocation("Local")
	fromDateParsed, _ := time.ParseInLocation("2006-01-02", fromDate, loc)
	toDateParsed, _ := time.ParseInLocation("2006-01-02", toDate, loc)

	// The filters should include access control for partner role
	filters := domain.PaymentHistoryFilters{
		FromDate:                  &fromDateParsed,
		ToDate:                    &toDateParsed,
		SortBy:                    "paid_date",
		SortOrder:                 "desc",
		Limit:                     10,
		Offset:                    0,
		EmployeeCreatedBy:         &userID, // Partner can only see employees they created
		EmployeeAssignedByPartner: &userID, // Or assigned to projects by them
	}

	mockTimesheetRepo.EXPECT().GetPaymentHistories(gomock.Any(), filters).Return([]*domain.PaymentHistory{}, int64(0), nil)

	_, err := service.GetPayrollHistories(context.Background(), request, userID, userRole)

	assert.NoError(t, err)
}

func TestPayrollService_GetPayrollHistories_InvalidDate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup mocks
	mockTimesheetRepo := mocks.NewMockTimesheetRepository(ctrl)

	service := &PayrollService{
		timesheetRepo: mockTimesheetRepo,
	}

	// Setup test data with invalid date format
	request := &dto.ListPayrollHistoriesRequest{
		FromDate: "invalid-date",
		Page:     1,
		PageSize: 10,
	}

	userID := uint(1)
	userRole := "admin"

	_, err := service.GetPayrollHistories(context.Background(), request, userID, userRole)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from_date")
}
