package employee

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/mocks"
	"github.com/golang/mock/gomock"
)

func TestGetEmployeeScheduleInfoReadyTarget(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeProfileService{ProjectEmployeeRepo: repo}
	project := checkInTargetProject(58, "LGD", []domain.GeofenceGate{{Name: "Cong C", Lat: 20.8679818, Lng: 106.5711738}})
	assignments := []*domain.ProjectEmployee{{
		ProjectID:       project.ID,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
		CheckInEnabled:  true,
	}}

	repo.EXPECT().GetByEmployee(gomock.Any(), uint(922)).Return(assignments, nil)
	repo.EXPECT().GetActiveProjectsForEmployee(gomock.Any(), uint(922)).Return([]*domain.Project{project}, nil)

	got := service.GetEmployeeScheduleInfo(context.Background(), 922)

	if got.CheckInTargetStatus != CheckInTargetReady {
		t.Fatalf("status = %q, want %q", got.CheckInTargetStatus, CheckInTargetReady)
	}
	if got.CheckInTarget == nil {
		t.Fatal("target is nil")
	}
	if got.CheckInTarget.ProjectID != 58 || got.CheckInTarget.ProjectName != "LGD" {
		t.Fatalf("target project = %#v", got.CheckInTarget)
	}
	if got.CheckInGeofenceRadiusMeters == nil || *got.CheckInGeofenceRadiusMeters != 300 {
		t.Fatalf("radius = %v, want 300", got.CheckInGeofenceRadiusMeters)
	}
}

func TestGetEmployeeScheduleInfoMissingGates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeProfileService{ProjectEmployeeRepo: repo}
	project := checkInTargetProject(58, "LGD", nil)
	assignments := []*domain.ProjectEmployee{{
		ProjectID:       project.ID,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
		CheckInEnabled:  true,
	}}

	repo.EXPECT().GetByEmployee(gomock.Any(), uint(922)).Return(assignments, nil)
	repo.EXPECT().GetActiveProjectsForEmployee(gomock.Any(), uint(922)).Return([]*domain.Project{project}, nil)

	got := service.GetEmployeeScheduleInfo(context.Background(), 922)

	if got.CheckInTargetStatus != CheckInTargetMissingGates {
		t.Fatalf("status = %q, want %q", got.CheckInTargetStatus, CheckInTargetMissingGates)
	}
	if got.CheckInTarget != nil {
		t.Fatalf("target = %#v, want nil", got.CheckInTarget)
	}
}

func TestGetEmployeeScheduleInfoAmbiguousBeforeFiltering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeProfileService{ProjectEmployeeRepo: repo}
	projectA := checkInTargetProject(58, "LGD", []domain.GeofenceGate{{Name: "Cong C", Lat: 20.8679818, Lng: 106.5711738}})
	projectB := checkInTargetProject(59, "Other", nil)
	assignments := []*domain.ProjectEmployee{{
		ProjectID:       projectA.ID,
		PaymentSchedule: string(domain.PaymentScheduleFlexible),
		CheckInEnabled:  true,
	}}

	repo.EXPECT().GetByEmployee(gomock.Any(), uint(922)).Return(assignments, nil)
	repo.EXPECT().GetActiveProjectsForEmployee(gomock.Any(), uint(922)).Return([]*domain.Project{projectA, projectB}, nil)

	got := service.GetEmployeeScheduleInfo(context.Background(), 922)

	if got.CheckInTargetStatus != CheckInTargetAmbiguous {
		t.Fatalf("status = %q, want %q", got.CheckInTargetStatus, CheckInTargetAmbiguous)
	}
	if got.CheckInTarget != nil {
		t.Fatalf("target = %#v, want nil", got.CheckInTarget)
	}
}

func checkInTargetProject(id uint, name string, gates []domain.GeofenceGate) *domain.Project {
	return &domain.Project{
		ID:                   id,
		Name:                 name,
		IsFlexible:           true,
		GeofenceRadiusMeters: 300,
		GeofenceGates:        gates,
	}
}
