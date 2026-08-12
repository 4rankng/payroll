package asynq

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	asynqlib "github.com/hibiken/asynq"

	"api-server/internal/config"
)

func TestHandleFlexPaySalaryNotificationRejectsInvalidPayload(t *testing.T) {
	h := &Handlers{}
	task := asynqlib.NewTask(TaskFlexPaySalaryNotification, []byte(`{"notification_id":0}`))
	if err := h.HandleFlexPaySalaryNotification(context.Background(), task); err != asynqlib.SkipRetry {
		t.Fatalf("error = %v, want SkipRetry", err)
	}
}

func TestHandleFlexPaySalaryNotificationSweepIsNilSafe(t *testing.T) {
	if err := (&Handlers{}).HandleFlexPaySalaryNotificationSweep(context.Background(), asynqlib.NewTask(TaskFlexPaySalaryNotificationSweep, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueFlexPaySalaryNotificationCanRecoverAfterArchive(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer redisServer.Close()

	client, err := NewClient(config.AsynqConfig{RedisAddr: redisServer.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.AsynqClient().Close() }()

	if err := client.EnqueueFlexPaySalaryNotification(73); err != nil {
		t.Fatal(err)
	}
	inspector := asynqlib.NewInspector(asynqlib.RedisClientOpt{Addr: redisServer.Addr()})
	pending, err := inspector.ListPendingTasks(QueueLow)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending = %d, err = %v", len(pending), err)
	}
	if err := inspector.ArchiveTask(QueueLow, pending[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := client.EnqueueFlexPaySalaryNotification(73); err != nil {
		t.Fatalf("re-enqueue after archive: %v", err)
	}
	pending, err = inspector.ListPendingTasks(QueueLow)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending after recovery = %d, err = %v", len(pending), err)
	}
}
