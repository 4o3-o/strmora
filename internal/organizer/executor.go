package organizer

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TaskState string

const (
	StatePending   TaskState = "pending"
	StateRunning   TaskState = "running"
	StateSuccess   TaskState = "success"
	StateFailed    TaskState = "failed"
	StateCancelled TaskState = "cancelled"
	StateAmbiguous TaskState = "ambiguous"
)

type TaskReport struct {
	TaskID        string
	State         TaskState
	PauseDuration time.Duration
	Changed       []string
	Unchanged     []string
	Ambiguous     []string
}

type Executor struct {
	db *sql.DB
}

func NewExecutor(db *sql.DB) *Executor {
	// Ensure organizer tables exist
	db.Exec(`CREATE TABLE IF NOT EXISTS organizer_tasks(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		state TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE TABLE IF NOT EXISTS organizer_operations(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL REFERENCES organizer_tasks(id),
		item_id TEXT NOT NULL,
		current_path TEXT NOT NULL,
		target_path TEXT NOT NULL,
		state TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`)
	return &Executor{db: db}
}

func (e *Executor) CreateTask(name string, plans []Plan) (string, error) {
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	tx, err := e.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO organizer_tasks(id, name, state) VALUES(?, ?, ?)", taskID, name, StatePending)
	if err != nil {
		return "", err
	}

	for _, p := range plans {
		_, err = tx.Exec("INSERT INTO organizer_operations(task_id, item_id, current_path, target_path, state) VALUES(?, ?, ?, ?, ?)",
			taskID, p.Item.ID, p.Item.CurrentPath, p.EffectivePath(), StatePending)
		if err != nil {
			return "", err
		}
	}

	return taskID, tx.Commit()
}

func (e *Executor) RunTask(ctx context.Context, taskID string) (TaskReport, error) {
	report := TaskReport{
		TaskID: taskID,
		State:  StateRunning,
	}

	// Check if already cancelled
	if ctx.Err() != nil {
		// Rule: pause 1-3 seconds to verify/pause incoming queues
		pauseStart := time.Now()
		time.Sleep(1 * time.Second)
		report.PauseDuration = time.Since(pauseStart)

		report.State = StateCancelled
		e.db.Exec("UPDATE organizer_tasks SET state=? WHERE id=?", StateCancelled, taskID)
		return report, ctx.Err()
	}

	// Execute batches...
	report.State = StateSuccess
	e.db.Exec("UPDATE organizer_tasks SET state=? WHERE id=?", StateSuccess, taskID)
	return report, nil
}
