package timesheet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkflow(t *testing.T) {
	workflow := NewWorkflow(nil)

	assert.NotNil(t, workflow)
}
