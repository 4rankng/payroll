package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *APIClient) SetToken(token string) {
	c.Token = token
}

// WithToken returns a shallow copy with a different token.
func (c *APIClient) WithToken(token string) *APIClient {
	return &APIClient{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Token:      token,
	}
}

// Login authenticates and stores the token.
func (c *APIClient) Login(username, password string) (*LoginResponse, error) {
	body := LoginRequest{Username: username, Password: password}
	var resp LoginResponse
	_, err := c.PostInto("/api/v1/auth/login", body, &resp)
	if err != nil {
		return nil, err
	}
	c.Token = resp.AccessToken
	return &resp, nil
}

// Get performs an authenticated GET and returns the raw API response.
func (c *APIClient) Get(path string) (*APIResponse, int, error) {
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// Post performs an authenticated POST and returns the raw API response.
func (c *APIClient) Post(path string, body any) (*APIResponse, int, error) {
	resp, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// GetInto performs GET and unmarshals data into dest.
func (c *APIClient) GetInto(path string, dest any) (*APIResponse, error) {
	apiResp, _, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	if apiResp.Status == "error" {
		return apiResp, fmt.Errorf("API error: %s", apiResp.Message)
	}
	if apiResp.Data != nil {
		resetDecodeTarget(dest)
		if err := json.Unmarshal(apiResp.Data, dest); err != nil {
			return apiResp, fmt.Errorf("unmarshal error: %w", err)
		}
	}
	return apiResp, nil
}

// PostInto performs POST and unmarshals data into dest.
func (c *APIClient) PostInto(path string, body any, dest any) (*APIResponse, error) {
	apiResp, statusCode, err := c.Post(path, body)
	if err != nil {
		return apiResp, err
	}
	if statusCode >= 400 {
		msg := apiResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", statusCode)
		}
		return apiResp, fmt.Errorf("API error (HTTP %d): %s", statusCode, msg)
	}
	if apiResp.Data != nil {
		resetDecodeTarget(dest)
		if err := json.Unmarshal(apiResp.Data, dest); err != nil {
			return apiResp, fmt.Errorf("unmarshal error: %w", err)
		}
	}
	return apiResp, nil
}

// PostExpectError performs POST and expects an error response. Returns the APIError and status.
func (c *APIClient) PostExpectError(path string, body any) (*APIError, int, error) {
	resp, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	var apiErr APIError
	if err := json.Unmarshal(respBody, &apiErr); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshal error response: %w (body: %s)", err, string(respBody))
	}
	return &apiErr, resp.StatusCode, nil
}

// GetExpectError performs GET and expects an error response.
func (c *APIClient) GetExpectError(path string) (*APIError, int, error) {
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	var apiErr APIError
	if err := json.Unmarshal(respBody, &apiErr); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshal error response: %w (body: %s)", err, string(respBody))
	}
	return &apiErr, resp.StatusCode, nil
}

// PollUntil calls path repeatedly until check returns true or timeout.
func (c *APIClient) PollUntil(path string, interval, timeout time.Duration, check func(*APIResponse) bool) (*APIResponse, error) {
	deadline := time.Now().Add(timeout)
	for {
		resp, _, err := c.Get(path)
		if err != nil {
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("poll timeout after %v: last error: %w", timeout, err)
			}
			time.Sleep(interval)
			continue
		}
		if check(resp) {
			return resp, nil
		}
		if time.Now().After(deadline) {
			return resp, fmt.Errorf("poll timeout after %v: condition not met", timeout)
		}
		time.Sleep(interval)
	}
}

func (c *APIClient) doRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return c.HTTPClient.Do(req)
}

func parseResponse(resp *http.Response) (*APIResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if len(body) == 0 {
		return &APIResponse{Status: "success"}, nil
	}

	apiResp := &APIResponse{}
	if err := json.Unmarshal(body, apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(body))
	}

	return apiResp, nil
}

// Put performs an authenticated PUT and returns the raw API response.
func (c *APIClient) Put(path string, body any) (*APIResponse, int, error) {
	resp, err := c.doRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// PutInto performs PUT and unmarshals data into dest.
func (c *APIClient) PutInto(path string, body any, dest any) (*APIResponse, error) {
	apiResp, statusCode, err := c.Put(path, body)
	if err != nil {
		return apiResp, err
	}
	if statusCode >= 400 {
		msg := apiResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", statusCode)
		}
		return apiResp, fmt.Errorf("API error (HTTP %d): %s", statusCode, msg)
	}
	if apiResp.Data != nil {
		resetDecodeTarget(dest)
		if err := json.Unmarshal(apiResp.Data, dest); err != nil {
			return apiResp, fmt.Errorf("unmarshal error: %w", err)
		}
	}
	return apiResp, nil
}

// Patch performs an authenticated PATCH and returns the raw API response.
func (c *APIClient) Patch(path string, body any) (*APIResponse, int, error) {
	resp, err := c.doRequest(http.MethodPatch, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// PatchInto performs PATCH and unmarshals data into dest.
func (c *APIClient) PatchInto(path string, body any, dest any) (*APIResponse, error) {
	apiResp, statusCode, err := c.Patch(path, body)
	if err != nil {
		return apiResp, err
	}
	if statusCode >= 400 {
		msg := apiResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", statusCode)
		}
		return apiResp, fmt.Errorf("API error (HTTP %d): %s", statusCode, msg)
	}
	if apiResp.Data != nil {
		resetDecodeTarget(dest)
		if err := json.Unmarshal(apiResp.Data, dest); err != nil {
			return apiResp, fmt.Errorf("unmarshal error: %w", err)
		}
	}
	return apiResp, nil
}

func resetDecodeTarget(dest any) {
	rv := reflect.ValueOf(dest)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
}

// Delete performs an authenticated DELETE and returns the raw API response.
func (c *APIClient) Delete(path string) (*APIResponse, int, error) {
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// PutExpectError performs PUT and expects an error response.
func (c *APIClient) PutExpectError(path string, body any) (*APIError, int, error) {
	resp, err := c.doRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	var apiErr APIError
	if err := json.Unmarshal(respBody, &apiErr); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshal error response: %w (body: %s)", err, string(respBody))
	}
	return &apiErr, resp.StatusCode, nil
}

// DeleteExpectError performs DELETE and expects an error response.
func (c *APIClient) DeleteExpectError(path string) (*APIError, int, error) {
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	var apiErr APIError
	if err := json.Unmarshal(respBody, &apiErr); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshal error response: %w (body: %s)", err, string(respBody))
	}
	return &apiErr, resp.StatusCode, nil
}

// DownloadFile sends a request and returns raw bytes for binary responses (Excel, ZIP, etc.).
func (c *APIClient) DownloadFile(method, path string, body any) ([]byte, http.Header, int, error) {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	// If JSON response, check for API error
	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var apiResp APIResponse
		if jsonErr := json.Unmarshal(respBody, &apiResp); jsonErr == nil && apiResp.Status == "error" {
			return nil, resp.Header, resp.StatusCode, fmt.Errorf("API error: %s", apiResp.Message)
		}
	}

	return respBody, resp.Header, resp.StatusCode, nil
}

// DownloadGet is a convenience method for GET binary downloads.
func (c *APIClient) DownloadGet(path string) ([]byte, http.Header, int, error) {
	return c.DownloadFile(http.MethodGet, path, nil)
}

// DownloadPost is a convenience method for POST binary downloads.
func (c *APIClient) DownloadPost(path string, body any) ([]byte, http.Header, int, error) {
	return c.DownloadFile(http.MethodPost, path, body)
}

// UploadBytes sends a multipart form file upload from in-memory bytes and returns the API response.
func (c *APIClient) UploadBytes(path, fileField, filename string, data []byte) (*APIResponse, int, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fileField, filename)
	if err != nil {
		return nil, 0, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, 0, fmt.Errorf("write data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, 0, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, &buf)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if strings.HasSuffix(path, "/timesheets/partner-import") {
		req.Header.Set("Idempotency-Key", fmt.Sprintf("integration-%d", time.Now().UnixNano()))
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}

// UploadFile sends a multipart form file upload and returns the API response.
func (c *APIClient) UploadFile(path, fileField, filePath string, fields map[string]string) (*APIResponse, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fileField, filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, 0, fmt.Errorf("copy file: %w", err)
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, 0, fmt.Errorf("write field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, 0, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, &buf)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if strings.HasSuffix(path, "/timesheets/partner-import") {
		req.Header.Set("Idempotency-Key", fmt.Sprintf("integration-%d", time.Now().UnixNano()))
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	apiResp, err := parseResponse(resp)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return apiResp, resp.StatusCode, nil
}
