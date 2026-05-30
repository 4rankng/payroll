package httputils

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultIDBase is the base used for parsing IDs (10 = decimal)
	DefaultIDBase = 10
	// IDBitSize is the bit size for parsing IDs (32 = uint32)
	IDBitSize = 32
)

// ParseIDParam parses a uint ID parameter from the gin context
// Returns error if the parameter is missing or invalid
func ParseIDParam(c *gin.Context, paramName string) (uint, error) {
	paramStr := c.Param(paramName)
	if paramStr == "" {
		return 0, fmt.Errorf("parameter %s is required", paramName)
	}

	id, err := strconv.ParseUint(paramStr, DefaultIDBase, IDBitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s must be a valid number", paramName, paramStr)
	}

	return uint(id), nil
}

// ParseQueryID parses a uint ID from query string
// Returns 0, nil if the parameter is missing (optional), error if invalid
func ParseQueryID(c *gin.Context, paramName string) (uint, error) {
	paramStr := c.Query(paramName)
	if paramStr == "" {
		return 0, nil // Optional parameter
	}

	id, err := strconv.ParseUint(paramStr, DefaultIDBase, IDBitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid %s query parameter: %s must be a valid number", paramName, paramStr)
	}

	return uint(id), nil
}

// ParseRequiredQueryID parses a uint ID from query string (required)
// Returns error if the parameter is missing or invalid
func ParseRequiredQueryID(c *gin.Context, paramName string) (uint, error) {
	paramStr := c.Query(paramName)
	if paramStr == "" {
		return 0, fmt.Errorf("query parameter %s is required", paramName)
	}

	id, err := strconv.ParseUint(paramStr, DefaultIDBase, IDBitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid %s query parameter: %s must be a valid number", paramName, paramStr)
	}

	return uint(id), nil
}

// ParseIDFromString parses a uint from a string
// Returns error if the string is empty or invalid
func ParseIDFromString(idStr string) (uint, error) {
	if idStr == "" {
		return 0, fmt.Errorf("ID string cannot be empty")
	}

	id, err := strconv.ParseUint(idStr, DefaultIDBase, IDBitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid ID format: %s must be a valid number", idStr)
	}

	return uint(id), nil
}
