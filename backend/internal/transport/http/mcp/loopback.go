package mcptransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxResponseBytes caps what one loopback call may return to the agent.
const maxResponseBytes = 1 << 20 // 1 MiB

// denyPrefixes are the /api/v1 surfaces a machine key may never reach through
// MCP: authentication, credential management, inbound webhooks, the MCP
// endpoint itself (recursion), and the vendor key surface (a different key
// type — it would 401 anyway; deny with a clear message instead).
var denyPrefixes = []string{
	"/api/v1/auth",
	"/api/v1/admin/api-keys",
	"/api/v1/admin/mcp-api-keys",
	"/api/v1/webhooks",
	"/api/v1/mcp",
	"/api/v1/vendor",
}

// denyPathMessage is returned to the agent for a guarded path.
const denyPathMessage = "Đường dẫn không được phép truy cập qua MCP"

// checkLoopbackPath enforces the guard before any call is dispatched.
func checkLoopbackPath(path string) error {
	if !strings.HasPrefix(path, "/api/v1/") {
		return fmt.Errorf("%s", denyPathMessage)
	}
	for _, prefix := range denyPrefixes {
		if strings.HasPrefix(path, prefix) {
			return fmt.Errorf("%s", denyPathMessage)
		}
	}
	return nil
}

// doLoopback replays one REST call against the real engine in-process. The
// request carries the caller's X-API-Key header, so every loopback call is
// re-authenticated and re-authorized by the real middleware chain — the MCP
// layer adds no authority of its own.
//
// The response body is returned verbatim (capped at 1 MiB); the key is never
// part of the returned text.
func doLoopback(engine *gin.Engine, apiKey, method, path, query string, body any) (int, string, error) {
	if err := checkLoopbackPath(path); err != nil {
		return 0, "", err
	}

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, "", fmt.Errorf("không mã hoá được nội dung: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	url := "http://payroll.internal" + path
	if query != "" {
		url += "?" + query
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("X-API-Key", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	respBody := rec.Body.String()
	if len(respBody) > maxResponseBytes {
		respBody = respBody[:maxResponseBytes] + "\n…[đã cắt]"
	}
	return rec.Code, respBody, nil
}
