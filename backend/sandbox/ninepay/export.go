package ninepay

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var exportTimeLocation = mustLoadVNLocation()

func mustLoadVNLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return loc
}

type exportRequestResponseEnvelope struct {
	Message string                 `json:"message"`
	Code    int                    `json:"code"`
	Data    exportRequestPayloadVO `json:"data"`
}

type exportRequestPayloadVO struct {
	Status   bool   `json:"status"`
	FileName string `json:"file_name"`
}

type indexResponseEnvelope struct {
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    indexPageVO `json:"data"`
}

type indexPageVO struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
}

const exportPerPage = 20

func (s *Server) exportIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params := readQuery(r)
	if err := s.verifySignature(r, params); err != nil {
		log.Printf("[9pay] export index: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	if dateFrom == "" || dateTo == "" {
		writeJSON(w, http.StatusOK, indexResponseEnvelope{
			Message: "date_from và date_to là bắt buộc",
			Code:    1,
			Data:    indexPageVO{},
		})
		return
	}
	from, err := time.ParseInLocation("02/01/2006", dateFrom, exportTimeLocation)
	if err != nil {
		writeJSON(w, http.StatusOK, indexResponseEnvelope{
			Message: "date_from phải là DD/MM/YYYY",
			Code:    1,
			Data:    indexPageVO{},
		})
		return
	}
	to, err := time.ParseInLocation("02/01/2006", dateTo, exportTimeLocation)
	if err != nil {
		writeJSON(w, http.StatusOK, indexResponseEnvelope{
			Message: "date_to phải là DD/MM/YYYY",
			Code:    1,
			Data:    indexPageVO{},
		})
		return
	}
	to = to.Add(24*time.Hour - time.Second)

	total := 0
	if s.poller != nil && s.poller.db != nil {
		count, err := countWalletPayments(r.Context(), s.poller.db, from, to)
		if err != nil {
			log.Printf("[9pay] export index: count: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		total = count
	}

	lastPage := 0
	if total > 0 {
		lastPage = (total + exportPerPage - 1) / exportPerPage
	}

	log.Printf("[9pay] export index: date_from=%s date_to=%s total=%d last_page=%d",
		dateFrom, dateTo, total, lastPage)

	writeJSON(w, http.StatusOK, indexResponseEnvelope{
		Message: "Lấy dữ liệu thành công",
		Code:    0,
		Data: indexPageVO{
			CurrentPage: 1,
			LastPage:    lastPage,
			Total:       total,
			PerPage:     exportPerPage,
		},
	})
}

func countWalletPayments(ctx context.Context, db *sql.DB, from, to time.Time) (int, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM provider_transactions WHERE created_at BETWEEN ? AND ?`,
		from, to).Scan(&count)
	return count, err
}

func (s *Server) requestExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params := readQuery(r)
	if err := s.verifySignature(r, params); err != nil {
		log.Printf("[9pay] export request: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	if dateFrom == "" || dateTo == "" {
		writeJSON(w, http.StatusOK, exportRequestResponseEnvelope{
			Message: "date_from và date_to là bắt buộc",
			Code:    1,
			Data:    exportRequestPayloadVO{Status: false},
		})
		return
	}
	from, err := time.ParseInLocation("02/01/2006", dateFrom, exportTimeLocation)
	if err != nil {
		writeJSON(w, http.StatusOK, exportRequestResponseEnvelope{
			Message: "date_from phải là DD/MM/YYYY",
			Code:    1,
			Data:    exportRequestPayloadVO{Status: false},
		})
		return
	}
	to, err := time.ParseInLocation("02/01/2006", dateTo, exportTimeLocation)
	if err != nil {
		writeJSON(w, http.StatusOK, exportRequestResponseEnvelope{
			Message: "date_to phải là DD/MM/YYYY",
			Code:    1,
			Data:    exportRequestPayloadVO{Status: false},
		})
		return
	}
	if from.After(to) {
		writeJSON(w, http.StatusOK, exportRequestResponseEnvelope{
			Message: "date_from phải <= date_to",
			Code:    1,
			Data:    exportRequestPayloadVO{Status: false},
		})
		return
	}

	pageStr := q.Get("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	fileName := fmt.Sprintf("transaction_disbursement_%s_%s_p%d_%d",
		from.Format("02012006"),
		to.Format("02012006"),
		page,
		time.Now().UnixMilli(),
	)
	log.Printf("[9pay] export request: date_from=%s date_to=%s page=%d file_name=%s", dateFrom, dateTo, page, fileName)

	writeJSON(w, http.StatusOK, exportRequestResponseEnvelope{
		Message: "Lấy dữ liệu thành công",
		Code:    0,
		Data: exportRequestPayloadVO{
			Status:   true,
			FileName: fileName,
		},
	})
}

func (s *Server) downloadExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params := readQuery(r)
	if err := s.verifySignature(r, params); err != nil {
		log.Printf("[9pay] export download: signature rejected: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fileName := r.URL.Query().Get("fileName")
	if fileName == "" {
		http.Error(w, "fileName is required", http.StatusBadRequest)
		return
	}
	from, to, err := parseFileNameDates(fileName)
	if err != nil {
		log.Printf("[9pay] export download: parse file_name=%q: %v", fileName, err)
		http.Error(w, "invalid fileName", http.StatusBadRequest)
		return
	}

	if s.poller == nil || s.poller.db == nil {
		log.Printf("[9pay] export download: no DB configured; returning header-only CSV")
		writeCSV(w, fileName, nil)
		return
	}

	rows, err := queryWalletPaymentsForExport(r.Context(), s.poller.db, from, to)
	if err != nil {
		log.Printf("[9pay] export download: query: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	log.Printf("[9pay] export download: file_name=%s range=%s..%s rows=%d",
		fileName, from.Format("2006-01-02"), to.Format("2006-01-02"), len(rows))

	writeCSV(w, fileName, rows)
}

func parseFileNameDates(name string) (time.Time, time.Time, error) {
	parts := strings.Split(name, "_")
	if len(parts) < 5 {
		return time.Time{}, time.Time{}, fmt.Errorf("unexpected token count")
	}
	fromStr, toStr := parts[2], parts[3]
	from, err := time.ParseInLocation("02012006", fromStr, exportTimeLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from: %w", err)
	}
	to, err := time.ParseInLocation("02012006", toStr, exportTimeLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to: %w", err)
	}
	to = to.Add(24 * time.Hour).Add(-time.Second)
	return from, to, nil
}

type providerTransactionRow struct {
	CreatedAt          time.Time
	UpdatedAt          time.Time
	InvoiceNo          sql.NullString
	RequestID          string
	RequestedAmount    int64
	Fee                int64
	RecipientBank      string
	RecipientAccountNo string
	RecipientName      string
	Description        sql.NullString
	Status             string
	ErrorMessage       sql.NullString
}

func queryWalletPaymentsForExport(ctx context.Context, db *sql.DB, from, to time.Time) ([]providerTransactionRow, error) {
	const q = `
		SELECT created_at, updated_at, invoice_no, request_id, requested_amount,
		       COALESCE(fee, 0), recipient_bank, recipient_account_no,
		       recipient_name, JSON_EXTRACT(metadata, '$.description'),
		       status, error_message
		FROM   provider_transactions
		WHERE  created_at BETWEEN ? AND ?
		ORDER  BY created_at ASC`
	rows, err := db.QueryContext(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []providerTransactionRow
	for rows.Next() {
		var r providerTransactionRow
		if err := rows.Scan(
			&r.CreatedAt, &r.UpdatedAt, &r.InvoiceNo, &r.RequestID,
			&r.RequestedAmount, &r.Fee, &r.RecipientBank, &r.RecipientAccountNo,
			&r.RecipientName, &r.Description, &r.Status, &r.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func writeCSV(w http.ResponseWriter, fileName string, rows []providerTransactionRow) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, fileName))
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	header := []string{
		"Thời gian khởi tạo", "Thời gian cập nhật", "Mã giao dịch", "Mã yêu cầu",
		"Loại giao dịch", "Giá trị giao dịch", "Phí", "Ngân hàng nhận tiền",
		"Tài khoản nhận tiền", "Tên chủ tài khoản", "Nội dung chi hộ", "Trạng thái",
	}
	if err := cw.Write(header); err != nil {
		log.Printf("[9pay] csv write header: %v", err)
		return
	}
	for _, r := range rows {
		invoice := ""
		if r.InvoiceNo.Valid {
			invoice = r.InvoiceNo.String
		}
		description := ""
		if r.Description.Valid {
			description = strings.Trim(r.Description.String, `"`)
		}
		errMsg := ""
		if r.ErrorMessage.Valid {
			errMsg = r.ErrorMessage.String
		}
		record := []string{
			r.CreatedAt.In(exportTimeLocation).Format("15:04:05 02/01/2006"),
			r.UpdatedAt.In(exportTimeLocation).Format("15:04:05 02/01/2006"),
			"'" + invoice, "'" + r.RequestID, "Chuyển tiền",
			strconv.FormatInt(r.RequestedAmount, 10), strconv.FormatInt(r.Fee, 10),
			r.RecipientBank, "'" + r.RecipientAccountNo, r.RecipientName,
			description, renderStatus(r.Status, errMsg),
		}
		if err := cw.Write(record); err != nil {
			log.Printf("[9pay] csv write row: %v", err)
			return
		}
	}
}

func renderStatus(status, errMsg string) string {
	switch status {
	case "completed":
		return "Thành công"
	case "failed":
		return "Thất bại | " + errMsg
	case "pending":
		return "Đang xử lý"
	case "verifying":
		return "Đang xác thực"
	case "reversed":
		return "Đã hoàn về"
	default:
		return status
	}
}

func readQuery(r *http.Request) []OrderedParam {
	q := r.URL.Query()
	out := make([]OrderedParam, 0, len(q))
	for k, v := range q {
		val := ""
		if len(v) > 0 {
			val = v[0]
		}
		out = append(out, OrderedParam{Key: k, Value: val})
	}
	return out
}
