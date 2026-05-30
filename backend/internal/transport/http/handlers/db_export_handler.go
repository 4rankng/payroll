package handlers

import (
	"net/http"
	"os"

	dbExportSvc "api-server/internal/app/services/db_export"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// DBExportHandler handles database export requests
type DBExportHandler struct {
	service      *dbExportSvc.DBExportService
	auditService *infrastructure.AuditService
}

// NewDBExportHandler creates a new DBExportHandler
func NewDBExportHandler(service *dbExportSvc.DBExportService, auditService *infrastructure.AuditService) *DBExportHandler {
	return &DBExportHandler{service: service, auditService: auditService}
}

// StartExport godoc
// POST /api/v1/db-export
// Starts an async full-database export job. Returns the job ID immediately.
func (h *DBExportHandler) StartExport(c *gin.Context) {
	userIDVal, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Unauthorized(c, constants.MsgUserNotAuthenticatedVN)
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c, constants.MsgUserNotAuthenticatedVN)
		return
	}

	jobID, err := h.service.CreateJob(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create export job"})
		return
	}

	// Start the export asynchronously
	h.service.StartExport(c.Request.Context(), jobID)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Export job started. Use the job_id to track progress.",
	})
}

// GetExportStatus godoc
// GET /api/v1/db-export/:jobId/status
// Returns the current status of an export job.
func (h *DBExportHandler) GetExportStatus(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	job, err := h.service.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListExportJobs godoc
// GET /api/v1/db-export
// Returns all known export jobs (for recovery after page reload).
func (h *DBExportHandler) ListExportJobs(c *gin.Context) {
	jobs, err := h.service.ListPendingJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	if jobs == nil {
		jobs = []*dbExportSvc.ExportJobStatus{}
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// DownloadExport godoc
// GET /api/v1/db-export/:jobId/download
// Downloads the completed export file.
func (h *DBExportHandler) DownloadExport(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	job, err := h.service.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	if job.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "export not yet completed", "status": job.Status})
		return
	}

	filePath, err := h.service.GetFilePath(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export file not found"})
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "export file has been removed"})
		return
	}

	// Emit audit event for database export download (non-blocking)
	if h.auditService != nil {
		go func() {
			// Use DoneCount as record count (number of tables exported)
			_ = h.auditService.LogFileExport(c.Request.Context(), "full_database", job.DoneCount, job.Filename)
		}()
	}

	c.Header("Content-Disposition", "attachment; filename=\""+job.Filename+"\"")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.File(filePath)
}
