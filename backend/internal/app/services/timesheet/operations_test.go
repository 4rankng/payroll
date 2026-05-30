package timesheet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOperations(t *testing.T) {
	ops := NewOperations(nil, nil, nil)

	assert.NotNil(t, ops)
}
