package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestContext(method, url string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	c.Request, _ = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	return c, w
}

func TestBindJSON_Failure(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name" binding:"required"`
	}

	// Invalid JSON
	c, w := setupTestContext("POST", "/test", nil)
	c.Request.Body = http.NoBody

	var req TestRequest
	result := BindJSON(c, &req)

	if result {
		t.Error("BindJSON() should return false for invalid request")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBindJSONWithError_CustomMessage(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name" binding:"required"`
	}

	c, w := setupTestContext("POST", "/test", nil)
	c.Request.Body = http.NoBody

	var req TestRequest
	customMsg := "Custom error message"
	result := BindJSONWithError(c, &req, customMsg)

	if result {
		t.Error("BindJSONWithError() should return false for invalid request")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBindJSONWithValidation_Success(t *testing.T) {
	type TestRequest struct {
		Age int `json:"age"`
	}

	requestBody := TestRequest{Age: 25}
	c, _ := setupTestContext("POST", "/test", requestBody)

	var req TestRequest
	result := BindJSONWithValidation(c, &req, func() error {
		if req.Age < 18 {
			return errors.New("age must be 18 or older")
		}
		return nil
	})

	if !result {
		t.Error("BindJSONWithValidation() should return true for valid request")
	}
}

func TestBindJSONWithValidation_ValidationFailure(t *testing.T) {
	type TestRequest struct {
		Age int `json:"age"`
	}

	requestBody := TestRequest{Age: 15}
	c, w := setupTestContext("POST", "/test", requestBody)

	var req TestRequest
	result := BindJSONWithValidation(c, &req, func() error {
		if req.Age < 18 {
			return errors.New("age must be 18 or older")
		}
		return nil
	})

	if result {
		t.Error("BindJSONWithValidation() should return false when validation fails")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBindJSONWithValidation_NilValidator(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name"`
	}

	requestBody := TestRequest{Name: "Test"}
	c, _ := setupTestContext("POST", "/test", requestBody)

	var req TestRequest
	result := BindJSONWithValidation(c, &req, nil)

	if !result {
		t.Error("BindJSONWithValidation() should return true with nil validator")
	}
}

func TestBindQuery_Success(t *testing.T) {
	type QueryParams struct {
		Page     int    `form:"page"`
		PageSize int    `form:"pageSize"`
		Search   string `form:"search"`
	}

	c, _ := setupTestContext("GET", "/test?page=1&pageSize=10&search=test", nil)

	var params QueryParams
	result := BindQuery(c, &params)

	if !result {
		t.Error("BindQuery() should return true for valid query params")
	}

	if params.Page != 1 {
		t.Errorf("Page = %d, want 1", params.Page)
	}

	if params.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", params.PageSize)
	}

	if params.Search != "test" {
		t.Errorf("Search = %s, want test", params.Search)
	}
}

func TestBindURI_Success(t *testing.T) {
	type URIParams struct {
		ID uint `uri:"id" binding:"required"`
	}

	c, _ := setupTestContext("GET", "/test/123", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "123"},
	}

	var params URIParams
	result := BindURI(c, &params)

	if !result {
		t.Error("BindURI() should return true for valid URI params")
	}

	if params.ID != 123 {
		t.Errorf("ID = %d, want 123", params.ID)
	}
}

func TestBindURI_Failure(t *testing.T) {
	type URIParams struct {
		ID uint `uri:"id" binding:"required"`
	}

	c, w := setupTestContext("GET", "/test/invalid", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid"},
	}

	var params URIParams
	result := BindURI(c, &params)

	if result {
		t.Error("BindURI() should return false for invalid URI params")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
