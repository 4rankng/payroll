package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmployeeUser_TableName(t *testing.T) {
	eu := EmployeeUser{}
	tableName := eu.TableName()
	assert.Equal(t, "employee_users", tableName)
}
