package ninepay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Reconciliation export endpoints. The 9pay merchant-web (BE) host
// serves these — distinct from the payment host that handles
// disbursement create / check-account / balance. In production:
// be.9pay.vn vs payment.9pay.vn; sandbox: sand-be.9pay.vn vs
// sand-payment.9pay.vn. The web host is configured separately on
// the Config struct (WebEndpoint).
//
// Auth model — IMPORTANT: these endpoints DO NOT use the HMAC
// signature scheme used by the disbursement payment API. They
// authenticate with `Authorization: Bearer <jwt>` where the JWT is
// obtained from the merchant-portal OAuth flow at AccountBaseURL.
// See portal_auth.go.
const (
	pathExportIndex    = "/api/transaction/disbursement/index"
	pathExportRequest  = "/api/transaction/disbursement/export"
	pathExportDownload = "/api/transaction/download"
)

// exportRequestResponse mirrors the JSON shape 9pay returns from
// GET /api/transaction/disbursement/export. The interesting field is
// data.file_name, which the caller must round-trip into the download
// endpoint to fetch the CSV bytes.
//
//	{
//	  "message": "Lấy dữ liệu thành công",
//	  "code": 0,
//	  "data": {
//	    "status": true,
//	    "file_name": "transaction_disbursement_01052026_07052026_177816930062"
//	  }
//	}
type exportRequestResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    struct {
		Status   bool   `json:"status"`
		FileName string `json:"file_name"`
	} `json:"data"`
}

// indexResponse mirrors the JSON shape returned by
// GET /api/transaction/disbursement/index. The pagination fields
// (last_page, total, per_page) tell the caller how many export pages
// to request.
//
//	{
//	  "message": "Lấy dữ liệu thành công",
//	  "code": 0,
//	  "data": {
//	    "current_page": 1,
//	    "last_page": 3,
//	    "total": 59,
//	    "per_page": 20
//	  }
//	}
type indexResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		Total       int `json:"total"`
		PerPage     int `json:"per_page"`
	} `json:"data"`
}

// GetExportPageCount calls the 9pay disbursement index endpoint to
// discover the total number of export pages for the given date range.
// Returns lastPage (>= 1 when data exists, 0 when empty).
func (c *Client) GetExportPageCount(ctx context.Context, dateFrom, dateTo time.Time) (int, error) {
	q := url.Values{}
	q.Set("date_from", dateFrom.Format("02/01/2006"))
	q.Set("date_to", dateTo.Format("02/01/2006"))
	q.Set("merchant_id", c.cfg.WebMerchantID)
	q.Set("order_code", "")
	q.Set("account_no", "")
	q.Set("account_name", "")
	q.Set("status", "")
	q.Set("mid_code", "")

	c.logger.Info("ninepay: get export page count",
		"date_from", q.Get("date_from"), "date_to", q.Get("date_to"))

	respBody, err := c.doBearerWebRequest(ctx, http.MethodGet, pathExportIndex, q)
	if err != nil {
		return 0, err
	}

	var out indexResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return 0, fmt.Errorf("ninepay: decode index response: %w (body: %s)",
			err, truncate(string(respBody), 256))
	}
	if out.Code != 0 {
		return 0, fmt.Errorf("ninepay: index rejected: code=%d message=%q",
			out.Code, out.Message)
	}
	if out.Data.LastPage == 0 || out.Data.Total == 0 {
		return 0, nil
	}
	return out.Data.LastPage, nil
}

// RequestExport asks 9pay to materialize a disbursement reconciliation
// CSV for [dateFrom, dateTo] inclusive for the given page. Returns the
// provider-issued fileName the caller must pass to DownloadExport.
//
// Dates are formatted as DD/MM/YYYY per the 9pay docs example
// (?date_from=01/05/2026). Both bounds use the merchant's local
// (Asia/Ho_Chi_Minh) day boundaries — 9pay does not accept a timezone.
func (c *Client) RequestExport(ctx context.Context, dateFrom, dateTo time.Time, page int) (string, error) {
	q := url.Values{}
	q.Set("date_from", dateFrom.Format("02/01/2006"))
	q.Set("date_to", dateTo.Format("02/01/2006"))
	// merchant_id here is the NUMERIC 9pay merchant ID (e.g. 2535), NOT
	// the alphanumeric MerchantKey used to sign the disbursement payment
	// API. Sending the wrong value silently filters all rows out — the
	// CSV comes back with only a header. Empty is treated as "all
	// merchants visible to the portal user" by 9pay, which is what we
	// want unless the operator scopes via NINEPAY_WEB_MERCHANT_ID.
	q.Set("merchant_id", c.cfg.WebMerchantID)
	q.Set("order_code", "")
	q.Set("account_no", "")
	q.Set("account_name", "")
	q.Set("status", "")
	q.Set("mid_code", "")
	q.Set("page", strconv.Itoa(page))
	q.Set("lang", "vi")

	c.logger.Info("ninepay: request export",
		"date_from", q.Get("date_from"), "date_to", q.Get("date_to"), "page", page)

	respBody, err := c.doBearerWebRequest(ctx, http.MethodGet, pathExportRequest, q)
	if err != nil {
		return "", err
	}

	var out exportRequestResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("ninepay: decode export response: %w (body: %s)",
			err, truncate(string(respBody), 256))
	}
	if out.Code != 0 || !out.Data.Status || out.Data.FileName == "" {
		return "", fmt.Errorf("ninepay: export rejected: code=%d status=%v message=%q",
			out.Code, out.Data.Status, out.Message)
	}
	return out.Data.FileName, nil
}

// DownloadExport fetches the CSV bytes for a fileName previously returned
// by RequestExport. Returns the raw CSV body — the caller is responsible
// for persisting and serving them.
func (c *Client) DownloadExport(ctx context.Context, fileName string) ([]byte, error) {
	if fileName == "" {
		return nil, fmt.Errorf("ninepay: download export: file name is empty")
	}
	q := url.Values{}
	q.Set("fileName", fileName)

	c.logger.Info("ninepay: download export", "file_name", fileName)

	body, err := c.doBearerWebRequest(ctx, http.MethodGet, pathExportDownload, q)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// doBearerWebRequest dispatches a request to the merchant-web host with
// an OAuth bearer token from the cached PortalAuth. Returns the raw
// response body — JSON or CSV depending on the endpoint.
//
// The local 9pay-mock does not enforce auth, so when c.portal is nil
// (OAuth not configured) the request is sent without an Authorization
// header. Production calls MUST have OAuth configured or this will
// fail at 9pay's edge.
func (c *Client) doBearerWebRequest(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	host := c.cfg.WebEndpoint
	if host == "" {
		return nil, fmt.Errorf("ninepay: web endpoint is not configured")
	}
	host = strings.TrimRight(host, "/")

	target := host + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return nil, fmt.Errorf("ninepay: build web request: %w", err)
	}
	req.Header.Set("Accept", "*/*")

	if c.portal != nil {
		token, err := c.portal.GetBearerToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("ninepay: get bearer token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		c.logger.Warn("ninepay: merchant-portal OAuth not configured; calling web endpoint without bearer token (works against local mock only)")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ninepay: http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ninepay: read web body: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		// Invalidate the cached token so the next call refreshes. 9pay
		// JWTs typically last 7200s; an unexpected 401 means the token
		// was revoked, the merchant rotated credentials, or the auth
		// flow itself broke.
		if c.portal != nil {
			c.portal.invalidate()
		}
		return nil, fmt.Errorf("ninepay: web 401 unauthorized: %s", truncate(string(body), 256))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ninepay: unexpected status %d: %s",
			resp.StatusCode, truncate(string(body), 256))
	}
	return body, nil
}
