package user

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUserService(t *testing.T) {
	hashSecret := "test-secret"
	hashSalt := "test-salt"

	service := NewUserService(nil, nil, nil, hashSecret, hashSalt)

	assert.NotNil(t, service)
	assert.Equal(t, hashSecret, service.hashSecret)
	assert.Equal(t, hashSalt, service.hashSalt)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.passwordValidator)
	assert.NotNil(t, service.businessContext)
	assert.NotNil(t, service.metricsCalculator)
	assert.Equal(t, uint32(64*1024), service.hashConfig.Memory)
	assert.Equal(t, uint32(3), service.hashConfig.Iterations)
	assert.Equal(t, uint8(2), service.hashConfig.Parallelism)
}

func TestNewMetricsCalculator(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	calc := NewMetricsCalculator(nil, logger)

	assert.NotNil(t, calc)
	assert.NotNil(t, calc.logger)
}

func TestNewBusinessContextMapper_Integration(t *testing.T) {
	mapper := NewBusinessContextMapper()

	assert.NotNil(t, mapper)

	// Test getting supported entity types
	types := mapper.GetSupportedEntityTypes()
	assert.NotEmpty(t, types)

	// Test getting supported actions
	actions := mapper.GetSupportedActionsForEntity("user")
	assert.NotEmpty(t, actions)
}
