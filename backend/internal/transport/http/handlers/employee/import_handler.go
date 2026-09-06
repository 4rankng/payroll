package employee

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/storage"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"
)

// ImportHandler handles employee import operations
type ImportHandler struct {
	importService   *employee.ImportService
	progressService *infrastructure.EmployeeImportProgressService
	fileStorage     storage.FileStorage
	asynqClient     *asynqinfra.Client
	auditService    *infrastructure.AuditService
}

// NewImportHandler creates a new import handler
func NewImportHandler(
	importService *employee.ImportService,
	progressService *infrastructure.EmployeeImportProgressService,
	fileStorage storage.FileStorage,
	asynqClient *asynqinfra.Client,
	auditService *infrastructure.AuditService,
) *ImportHandler {
	return &ImportHandler{
		importService:   importService,
		progressService: progressService,
		fileStorage:     fileStorage,
		asynqClient:     asynqClient,
		auditService:    auditService,
	}
}

// PostImportEmployees handles the upload of employee import Excel files
// @Summary Import employees from Excel
// @Description Upload an Excel file to import employees asynchronously
// @Tags employees
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file (.xlsx)"
// @Success 202 {object} response.Response{data=dto.EmployeeImportResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /imports/employees [post]
func (h *ImportHandler) PostImportEmployees(c *gin.Context) {
	ctx := c.Request.Context()
	logger := observability.GetLogger()

	// Get authenticated user ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, "User not authenticated")
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		response.Forbidden(c, constants.MsgInvalidUserIDVN)
		return
	}

	// Guarded multipart parse: body cap, size cap, xlsx magic sniff
	// (replaces the former extension + 10MB checks).
	fileHeader, ok := uploadguard.Validate(c, false)
	if !ok {
		return
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		logger.Error("failed to open uploaded file", "error", err)
		response.InternalServerError(c, constants.MsgFailedToOpenFileVN)
		return
	}
	defer func() { _ = file.Close() }()

	storedFile, err := h.fileStorage.Store(file, fileHeader, "employee_imports")
	if err != nil {
		logger.Error("failed to store file", "error", err)
		response.InternalServerError(c, constants.MsgFailedToStoreFileVN)
		return
	}

	// Generate import ID
	importID := uuid.New().String()

	// Parse Excel to get row count
	// We'll need to read the file with excelize
	// For now, initialize with 0 and let worker update it
	totalRows := 0

	// Initialize import progress
	if err := h.progressService.StartImport(ctx, importID, totalRows, userIDUint, storedFile.FilePath); err != nil {
		logger.Error("failed to initialize import progress", "import_id", importID, "error", err)
		response.InternalServerError(c, constants.MsgFailedToInitImportVN)
		return
	}

	if err := h.asynqClient.EnqueueEmployeeImport(importID); err != nil {
		logger.Error("failed to enqueue import job", "import_id", importID, "error", err)
		response.InternalServerError(c, constants.MsgFailedToQueueImportJobVN)
		return
	}

	logger.Info("employee import job queued",
		"import_id", importID,
		"user_id", userIDUint,
		"filename", fileHeader.Filename,
		"file_path", storedFile.FilePath,
	)

	// Emit audit event for file import
	if h.auditService != nil {
		_ = h.auditService.LogFileImport(ctx, "employees", totalRows, fileHeader.Filename)
	}

	response.Success(c, dto.EmployeeImportResponse{
		ImportID: importID,
		Status:   "pending",
		Message:  "Import job queued successfully",
	}, "Import job queued successfully")
}

// GetImportStatus retrieves the status of an employee import job
// @Summary Get import status
// @Description Get the current status and progress of an employee import job
// @Tags employees
// @Produce json
// @Param id path string true "Import ID"
// @Success 200 {object} response.Response{data=dto.EmployeeImportStatus}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /imports/employees/{id}/status [get]
func (h *ImportHandler) GetImportStatus(c *gin.Context) {
	ctx := c.Request.Context()
	logger := observability.GetLogger()

	importID := c.Param("id")
	if importID == "" {
		response.BadRequest(c, "Import ID is required")
		return
	}

	status, err := h.progressService.GetImportStatus(ctx, importID)
	if err != nil {
		logger.Warn("import status not found", "import_id", importID, "error", err)
		response.NotFound(c, "Import not found")
		return
	}

	response.Success(c, status, "")
}

// Helper function for worker to get stored file path
func (h *ImportHandler) GetStoredFilePath(ctx context.Context, importID string) (string, uint, error) {
	progress, err := h.progressService.GetProgress(ctx, importID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get import progress: %w", err)
	}

	return progress.FilePath, progress.UserID, nil
}
