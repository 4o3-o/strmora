package organizer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"strmora/internal/database"
)

func TestExecutorTransactionAndCancellation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exec_test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec := NewExecutor(db)

	ctx, cancel := context.WithCancel(context.Background())

	// Create a task
	taskID, err := exec.CreateTask("organize-1", []Plan{
		{
			Item:       PlanItem{ID: "item-1", CurrentPath: "/src/A.mkv"},
			TargetPath: "/dst/A.mkv",
		},
		{
			Item:       PlanItem{ID: "item-2", CurrentPath: "/src/B.mkv"},
			TargetPath: "/dst/B.mkv",
		},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	// Trigger cancellation
	cancel()

	report, err := exec.RunTask(ctx, taskID)
	if err == nil && report.State != StateCancelled {
		t.Fatalf("expected task cancelled, got %s, err: %v", report.State, err)
	}
	if report.PauseDuration < 1*time.Second || report.PauseDuration > 3*time.Second {
		t.Fatalf("expected cancellation pause between 1s and 3s, got %v", report.PauseDuration)
	}
}
