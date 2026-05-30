package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectUser_TableName(t *testing.T) {
	pu := ProjectUser{}
	tableName := pu.TableName()
	assert.Equal(t, "project_users", tableName)
}
