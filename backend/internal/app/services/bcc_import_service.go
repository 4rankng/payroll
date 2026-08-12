package services

import (
	"errors"

	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	timesheetSvc "api-server/internal/app/services/timesheet"
	"api-server/internal/domain"
	"api-server/internal/infra/storage"

	"github.com/redis/go-redis/v9"
)

// BCCImportService imports timesheet workbooks (legacy BCC, multi-position,
// weekly BCC, weekly payment) into durable import jobs and processes them into
// timesheets. Its methods are split across sibling files by concern:
// bcc_import_lifecycle.go (accept/process lifecycle + job state),
// bcc_import_process.go (legacy BCC parse+import pipeline),
// bcc_import_replacement.go (timesheet replacement logic),
// bcc_import_result.go (result building + error mapping + audit),
// bcc_import_weekly.go (weekly BCC/payment uploads),
// bcc_import_multi_position.go, bcc_import_lock.go, bcc_import_types.go.
type BCCImportService struct {
	payrateRepo         domain.PayrateRepository
	timesheetReader     domain.TimesheetReader
	timesheetWriter     domain.TimesheetWriter
	timesheetService    *timesheetSvc.TimesheetService
	assetRepo           domain.AssetRepository
	fileStorage         storage.FileStorage
	transactionManager  domain.TransactionManager
	redis               *redis.Client
	employeeService     *employee.EmployeeService
	employeeUserService *employee.EmployeeUserService
	projectEmployeeSvc  *project.ProjectEmployeeService
	importJobRepo       domain.TimesheetImportJobRepository
	importEnqueuer      domain.TimesheetImportEnqueuer
}

var (
	ErrBCCIdempotencyConflict = errors.New("idempotency key đã được dùng cho một tệp khác")
	ErrBCCImportScopeBusy     = errors.New("dự án và tháng này đang có một tệp BCC được xử lý")
)

func NewBCCImportService(
	payrateRepo domain.PayrateRepository,
	timesheetRepo domain.TimesheetRepository,
	timesheetService *timesheetSvc.TimesheetService,
	assetRepo domain.AssetRepository,
	fileStorage storage.FileStorage,
	transactionManager domain.TransactionManager,
	redisClient *redis.Client,
	employeeService *employee.EmployeeService,
	employeeUserService *employee.EmployeeUserService,
	projectEmployeeSvc *project.ProjectEmployeeService,
	importJobRepo domain.TimesheetImportJobRepository,
	importEnqueuer domain.TimesheetImportEnqueuer,
) *BCCImportService {
	return &BCCImportService{
		payrateRepo:         payrateRepo,
		timesheetReader:     timesheetRepo,
		timesheetWriter:     timesheetRepo,
		timesheetService:    timesheetService,
		assetRepo:           assetRepo,
		fileStorage:         fileStorage,
		transactionManager:  transactionManager,
		redis:               redisClient,
		employeeService:     employeeService,
		employeeUserService: employeeUserService,
		projectEmployeeSvc:  projectEmployeeSvc,
		importJobRepo:       importJobRepo,
		importEnqueuer:      importEnqueuer,
	}
}
