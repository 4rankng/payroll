package employee

import (
	"context"
	"testing"

	"api-server/internal/app/services/audit"
	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
)

type discardEventBus struct{}

func (discardEventBus) Publish(context.Context, ...domain.DomainEvent) error { return nil }
func (discardEventBus) Subscribe(string, domain.EventHandler)                {}
func (discardEventBus) SubscribeAll(domain.EventHandler)                     {}

type recordingEventBus struct {
	events []domain.DomainEvent
}

func (b *recordingEventBus) Publish(_ context.Context, events ...domain.DomainEvent) error {
	b.events = append(b.events, events...)
	return nil
}
func (*recordingEventBus) Subscribe(string, domain.EventHandler) {}
func (*recordingEventBus) SubscribeAll(domain.EventHandler)      {}

type deletionAttendanceRepo struct {
	domain.AttendanceRepository
	hardDeletedEmployeeID uint
}

func (r *deletionAttendanceRepo) HardDeleteByEmployeeID(_ context.Context, employeeID uint) error {
	r.hardDeletedEmployeeID = employeeID
	return nil
}

func TestDeleteEmployeePermanentlyRemovesOnlyOperationalHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	timesheetRepo := mocks.NewMockTimesheetRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	attendanceRepo := &deletionAttendanceRepo{}
	service := &EmployeeService{
		EmployeeRepo:        employeeRepo,
		TimesheetRepo:       timesheetRepo,
		AttendanceRepo:      attendanceRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		TransactionManager:  directTransactionManager{},
		events:              discardEventBus{},
	}

	ctx := audit.WithAuditContext(context.Background(), 44, "Admin")
	employee := &domain.Employee{ID: 31, Fullname: "Nguyen Van A", CCCD: "031086019743"}
	employeeRepo.EXPECT().GetByIDForUpdate(ctx, uint(31)).Return(employee, nil)
	projectEmployeeRepo.EXPECT().HasActiveFlexiblePaymentScheduleByEmployeeID(ctx, uint(31)).Return(false, nil)
	timesheetRepo.EXPECT().HasProtectedTimesheetsByEmployeeID(ctx, uint(31)).Return(false, nil)
	projectEmployeeRepo.EXPECT().HardDeleteAssignmentsByEmployeeID(ctx, uint(31)).Return(nil)
	timesheetRepo.EXPECT().HardDeleteOperationalByEmployeeID(ctx, uint(31)).Return(nil)
	employeeRepo.EXPECT().HardDelete(ctx, uint(31)).Return(nil)

	result, err := service.DeleteEmployee(ctx, 31, 44)
	if err != nil {
		t.Fatalf("DeleteEmployee() error = %v", err)
	}
	if result.RetainedForFinancialHistory {
		t.Fatal("operational-history employee was retained")
	}
	if attendanceRepo.hardDeletedEmployeeID != 31 {
		t.Fatalf("attendance hard-deleted employee = %d, want 31", attendanceRepo.hardDeletedEmployeeID)
	}
}

func TestDeleteEmployeeRetainsProtectedHistoryAndUnlinksProjects(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	timesheetRepo := mocks.NewMockTimesheetRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	attendanceRepo := &deletionAttendanceRepo{}
	events := &recordingEventBus{}
	service := &EmployeeService{
		EmployeeRepo:        employeeRepo,
		TimesheetRepo:       timesheetRepo,
		AttendanceRepo:      attendanceRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		TransactionManager:  directTransactionManager{},
		events:              events,
	}

	ctx := audit.WithAuditContext(context.Background(), 44, "Admin")
	employee := &domain.Employee{ID: 32, Fullname: "Tran Thi B", CCCD: "031086019744"}
	employeeRepo.EXPECT().GetByIDForUpdate(ctx, uint(32)).Return(employee, nil)
	projectEmployeeRepo.EXPECT().HasActiveFlexiblePaymentScheduleByEmployeeID(ctx, uint(32)).Return(false, nil)
	timesheetRepo.EXPECT().HasProtectedTimesheetsByEmployeeID(ctx, uint(32)).Return(true, nil)
	projectEmployeeRepo.EXPECT().DeleteAssignmentsByEmployeeID(ctx, uint(32)).Return(nil)

	result, err := service.DeleteEmployee(ctx, 32, 44)
	if err != nil {
		t.Fatalf("DeleteEmployee() error = %v", err)
	}
	if !result.RetainedForFinancialHistory {
		t.Fatal("protected-history employee was hard deleted")
	}
	if len(events.events) != 1 || events.events[0].EventType() != "EmployeeProjectAssignmentsRemoved" {
		t.Fatal("unlinking an employee with protected history must emit its own audit event")
	}
	if attendanceRepo.hardDeletedEmployeeID != 0 {
		t.Fatalf("paid-history employee attendance was hard deleted: %d", attendanceRepo.hardDeletedEmployeeID)
	}
}

func TestDeleteEmployeeRejectsActiveFlexiblePaymentSchedule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{
		EmployeeRepo:        employeeRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		TransactionManager:  directTransactionManager{},
		events:              discardEventBus{},
	}

	ctx := audit.WithAuditContext(context.Background(), 44, "Admin")
	employee := &domain.Employee{ID: 33, Fullname: "Le Thi C", CCCD: "031086019745"}
	employeeRepo.EXPECT().GetByIDForUpdate(ctx, uint(33)).Return(employee, nil)
	projectEmployeeRepo.EXPECT().HasActiveFlexiblePaymentScheduleByEmployeeID(ctx, uint(33)).Return(true, nil)

	result, err := service.DeleteEmployee(ctx, 33, 44)
	if err == nil {
		t.Fatal("DeleteEmployee() succeeded for an employee on a flexible payment schedule")
	}
	if result != nil {
		t.Fatalf("DeleteEmployee() result = %#v, want nil", result)
	}
}
