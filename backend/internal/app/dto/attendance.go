package dto

import "time"

// CheckInRequest represents the request to check in.
// ProjectID is optional — when omitted, the server auto-detects the employee's
// active flexible project. When the employee belongs to multiple flexible
// projects, project_id must be provided.
type CheckInRequest struct {
	ProjectID uint    `json:"project_id"`
	Lat       float64 `json:"lat" binding:"required"`
	Lng       float64 `json:"lng" binding:"required"`
	Accuracy  float64 `json:"accuracy"` // GPS accuracy in meters; 0 = unknown
	GpsAt     int64   `json:"gps_at"`   // device GPS fix timestamp (epoch ms); 0 = unknown
	// Optional browser-side acquisition diagnostics. The service currently uses
	// Accuracy/GpsAt for enforcement and storage; these fields make the API
	// contract explicit for clients that warm up GPS with watchPosition.
	GpsSampleCount  int     `json:"gps_sample_count"`
	GpsBestAccuracy float64 `json:"gps_best_accuracy"`
	GpsElapsedMs    int64   `json:"gps_elapsed_ms"`
}

// CheckOutRequest represents the request to check out
type CheckOutRequest struct {
	Lat             float64 `json:"lat" binding:"required"`
	Lng             float64 `json:"lng" binding:"required"`
	Accuracy        float64 `json:"accuracy"` // GPS accuracy in meters; 0 = unknown
	GpsAt           int64   `json:"gps_at"`   // device GPS fix timestamp (epoch ms); 0 = unknown
	ConfirmNoSalary bool    `json:"confirm_no_salary"`
	GpsSampleCount  int     `json:"gps_sample_count"`
	GpsBestAccuracy float64 `json:"gps_best_accuracy"`
	GpsElapsedMs    int64   `json:"gps_elapsed_ms"`
}

// LogDeviceAttemptRequest records a device-level GPS failure (denied / timeout /
// unavailable / unsupported) so the admin can see that the worker tried but the
// device couldn't produce a fix.
type LogDeviceAttemptRequest struct {
	AttemptType string `json:"attempt_type" binding:"required,oneof=check_in check_out"`
	GpsStatus   string `json:"gps_status" binding:"required,oneof=denied timeout unavailable unsupported"`
}

// AttendanceResponse represents an attendance record
type AttendanceResponse struct {
	ID                 uint       `json:"id"`
	ProjectID          uint       `json:"project_id"`
	EmployeeID         uint       `json:"employee_id"`
	Date               time.Time  `json:"date"`
	CheckInTime        time.Time  `json:"check_in_time"`
	CheckInGate        string     `json:"check_in_gate"`
	CheckOutTime       *time.Time `json:"check_out_time,omitempty"`
	CheckOutGate       *string    `json:"check_out_gate,omitempty"`
	EarningAmount      *int64     `json:"earning_amount,omitempty"`
	SalaryRejectReason *string    `json:"salary_reject_reason,omitempty"`
	SalaryStatus       string     `json:"salary_status"`
	SalaryMessage      string     `json:"salary_message"`
	Status             string     `json:"status"`
}

// AdminAttendanceResponse represents the detailed attendance record for admin view
type AdminAttendanceResponse struct {
	ID                              uint       `json:"id"`
	ProjectID                       uint       `json:"project_id"`
	ProjectName                     string     `json:"project_name"`
	EmployeeID                      uint       `json:"employee_id"`
	EmployeeName                    string     `json:"employee_name"`
	Date                            time.Time  `json:"date"`
	CheckInTime                     time.Time  `json:"check_in_time"`
	CheckInLat                      float64    `json:"check_in_lat"`
	CheckInLng                      float64    `json:"check_in_lng"`
	CheckInAccuracy                 *float64   `json:"check_in_accuracy,omitempty"`
	CheckInGpsAt                    *time.Time `json:"check_in_gps_at,omitempty"`
	CheckInGate                     string     `json:"check_in_gate"`
	CheckOutTime                    *time.Time `json:"check_out_time,omitempty"`
	CheckOutLat                     *float64   `json:"check_out_lat,omitempty"`
	CheckOutLng                     *float64   `json:"check_out_lng,omitempty"`
	CheckOutAccuracy                *float64   `json:"check_out_accuracy,omitempty"`
	CheckOutGpsAt                   *time.Time `json:"check_out_gps_at,omitempty"`
	CheckOutGate                    *string    `json:"check_out_gate,omitempty"`
	EarningAmount                   *int64     `json:"earning_amount,omitempty"`
	SalaryRejectReason              *string    `json:"salary_reject_reason,omitempty"`
	RejectedAt                      *time.Time `json:"rejected_at,omitempty"`
	NearestCheckpointName           *string    `json:"nearest_checkpoint_name,omitempty"`
	NearestCheckpointLat            *float64   `json:"nearest_checkpoint_lat,omitempty"`
	NearestCheckpointLng            *float64   `json:"nearest_checkpoint_lng,omitempty"`
	NearestCheckpointDistanceMeters *float64   `json:"nearest_checkpoint_distance_meters,omitempty"`
	GeofenceRadiusMeters            *uint      `json:"geofence_radius_meters,omitempty"`
	Status                          string     `json:"status"`
}

// PaginatedAttendanceResponse represents a paginated list of attendances
type PaginatedAttendanceResponse struct {
	Data  []AdminAttendanceResponse `json:"data"`
	Total int64                     `json:"total"`
}

// AdminFailedAttemptResponse represents a single failed check-in/out attempt for the admin drill-down.
type AdminFailedAttemptResponse struct {
	ID                              uint       `json:"id"`
	EmployeeID                      uint       `json:"employee_id"`
	EmployeeName                    string     `json:"employee_name,omitempty"`
	AttemptType                     string     `json:"attempt_type"`
	ReasonCategory                  string     `json:"reason_category"`
	ProjectID                       uint       `json:"project_id"`
	Lat                             *float64   `json:"lat,omitempty"`
	Lng                             *float64   `json:"lng,omitempty"`
	Accuracy                        *float64   `json:"accuracy,omitempty"`
	GpsAt                           *time.Time `json:"gps_at,omitempty"`
	NearestCheckpointName           *string    `json:"nearest_checkpoint_name,omitempty"`
	NearestCheckpointLat            *float64   `json:"nearest_checkpoint_lat,omitempty"`
	NearestCheckpointLng            *float64   `json:"nearest_checkpoint_lng,omitempty"`
	NearestCheckpointDistanceMeters *float64   `json:"nearest_checkpoint_distance_meters,omitempty"`
	GeofenceRadiusMeters            *uint      `json:"geofence_radius_meters,omitempty"`
	ErrorMessage                    *string    `json:"error_message,omitempty"`
	CreatedAt                       time.Time  `json:"created_at"`
}

// PaginatedFailedAttemptResponse wraps a failed-attempt list with its total for pagination.
type PaginatedFailedAttemptResponse struct {
	Data  []AdminFailedAttemptResponse `json:"data"`
	Total int64                        `json:"total"`
}
