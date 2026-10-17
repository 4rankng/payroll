// Package mcptransport serves the MCP agent surface over Streamable HTTP. It
// exposes two tools — payroll_api and payroll_endpoints — that replay REST
// calls against the real gin engine in-process, so every call is authenticated
// and authorized by the same middleware chain as external traffic. The MCP
// layer adds no authority of its own.
package mcptransport

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"api-server/internal/app/services/mcpapikey"

	"github.com/gin-gonic/gin"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// shared caches JSON schemas across per-request servers so schema generation
// happens once, not per request.
var shared = &mcp.SchemaCache{}

// payrollAPIInput is the input schema for the payroll_api tool.
type payrollAPIInput struct {
	// Method is the HTTP verb: GET, POST, PUT, PATCH or DELETE.
	Method string `json:"method"`
	// Path is the REST path under /api/v1, e.g. /api/v1/employees.
	Path string `json:"path"`
	// Query is an optional raw query string, e.g. "page=1&page_size=20".
	Query string `json:"query,omitempty"`
	// Body is an optional JSON request body.
	Body map[string]any `json:"body,omitempty"`
}

// NewHandler builds the Streamable HTTP endpoint. The callback creates one
// server per request carrying that request's key: the transport is stateless,
// so every tool call is authorized with the key of ITS OWN request — a session
// can never lend its authority to a request authenticated with a different
// key. The key never leaves the request and is never logged.
//
// keys is consulted as a fail-closed guard even if the handler is ever mounted
// without the route middleware: an unauthenticated key yields a server with no
// tools.
func NewHandler(engine *gin.Engine, keys *mcpapikey.Service, logger *slog.Logger) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		apiKey := req.Header.Get("X-API-Key")
		if keys != nil {
			if _, _, err := keys.Authenticate(req.Context(), apiKey); err != nil {
				return mcp.NewServer(&mcp.Implementation{Name: "payroll", Version: "1.0.0"}, &mcp.ServerOptions{
					SchemaCache: shared,
					Logger:      logger,
				})
			}
		}
		server := mcp.NewServer(&mcp.Implementation{Name: "payroll", Version: "1.0.0"}, &mcp.ServerOptions{
			SchemaCache: shared,
			Logger:      logger,
		})
		registerTools(server, engine, apiKey)
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})
}

// registerTools adds the two tools bound to one caller's key.
func registerTools(server *mcp.Server, engine *gin.Engine, apiKey string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "payroll_api",
		Description: "Call any Payroll REST endpoint under /api/v1 with the configured API key. " +
			"READ ONLY keys may only use GET. Paths /api/v1/auth, credential management, " +
			"webhooks and /api/v1/vendor are not available. Responses are raw API JSON.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in payrollAPIInput) (*mcp.CallToolResult, any, error) {
		method := strings.ToUpper(strings.TrimSpace(in.Method))
		if !validMethod(method) {
			return toolText("HTTP 400\nmethod phải là một trong: GET, POST, PUT, PATCH, DELETE", true), nil, nil
		}
		path := strings.TrimSpace(in.Path)
		status, body, err := doLoopback(engine, apiKey, method, path, strings.TrimSpace(in.Query), in.Body)
		if err != nil {
			return toolText(err.Error(), true), nil, nil
		}
		return toolText("HTTP "+strconv.Itoa(status)+"\n"+body, status >= 400), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "payroll_endpoints",
		Description: "List every REST endpoint the configured key can reach, with its method.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return toolText(strings.Join(reachableEndpoints(engine), "\n"), false), nil, nil
	})
}

// validMethod reports whether m is an allowed REST verb.
func validMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// reachableEndpoints formats the engine's /api/v1 routes minus the guarded
// surfaces, one per line: "<METHOD> <path>  [read|mutation]".
func reachableEndpoints(engine *gin.Engine) []string {
	var lines []string
	seen := map[string]bool{}
	for _, route := range engine.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		if err := checkLoopbackPath(route.Path); err != nil {
			continue
		}
		line := route.Method + " " + route.Path + "  [" + breadth(route.Method) + "]"
		if !seen[line] {
			seen[line] = true
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return lines
}

// breadth labels one verb as read or mutation.
func breadth(method string) string {
	if method == http.MethodGet || method == http.MethodHead {
		return "read"
	}
	return "mutation"
}

// toolText builds the one-text payload every tool returns.
func toolText(text string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: isError,
	}
}
