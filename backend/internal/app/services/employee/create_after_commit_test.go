package employee

import (
	"context"
	"testing"
	"time"

	userservice "api-server/internal/app/services/user"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

type employeeCreationEventBus struct {
	createEmployeeTestEventBus
	created chan context.Context
}

func (b employeeCreationEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, event := range events {
		if event.EventType() == "EmployeeCreated" {
			b.created <- ctx
		}
	}
	return nil
}

func TestCreateEmployeePublishesOnlyAfterOuterCommit(t *testing.T) {
	for _, commit := range []bool{false, true} {
		name := "rollback"
		if commit {
			name = "commit"
		}
		t.Run(name, func(t *testing.T) {
			events := employeeCreationEventBus{created: make(chan context.Context, 1)}
			users := &createEmployeeTestUserRepo{}
			service := &EmployeeService{
				EmployeeRepo: &stubEmployeeRepo{}, UserRepo: users,
				UserService:        userservice.NewUserService(users, nil, events, "test-secret", "test-salt"),
				TransactionManager: directTransactionManager{}, events: events,
			}
			transaction := &domain.TransactionContext{IsTransactional: true}
			ctx, cancel := context.WithCancel(domain.WithTransactionContext(context.Background(), transaction))
			_, err := service.CreateEmployeeFromImport(ctx, &domain.Employee{
				Fullname: "Synthetic Import Employee", CCCD: "099260900502",
			}, 99)
			require.NoError(t, err)
			cancel()
			select {
			case <-events.created:
				t.Fatal("employee event published before outer transaction committed")
			default:
			}
			if !commit {
				return
			}
			transaction.RunAfterCommitCallbacks()
			select {
			case eventCtx := <-events.created:
				_, transactional := domain.GetTransactionFromContext(eventCtx)
				require.False(t, transactional, "consumer must not reuse the committed transaction")
				require.NoError(t, eventCtx.Err(), "consumer must outlive the import context")
			case <-time.After(time.Second):
				t.Fatal("employee event missing after commit")
			}
		})
	}
}
