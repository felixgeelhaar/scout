package browse

import (
	"errors"
	"testing"
)

func TestTaskLifecycleHappyPath(t *testing.T) {
	tracker, err := NewTaskTracker("test-task")
	if err != nil {
		t.Fatalf("NewTaskTracker: %v", err)
	}
	defer tracker.Stop()

	if tracker.State() != StatePending {
		t.Errorf("expected pending, got %s", tracker.State())
	}

	tracker.Start()
	if tracker.State() != StateRunning {
		t.Errorf("expected running, got %s", tracker.State())
	}
	if tracker.Context().Attempt != 1 {
		t.Errorf("expected attempt=1, got %d", tracker.Context().Attempt)
	}

	tracker.Success()
	if tracker.State() != StateSuccess {
		t.Errorf("expected success, got %s", tracker.State())
	}
	if !tracker.IsDone() {
		t.Error("expected done after success")
	}
}

func TestTaskLifecycleFailAndReset(t *testing.T) {
	tracker, err := NewTaskTracker("fail-task")
	if err != nil {
		t.Fatalf("NewTaskTracker: %v", err)
	}
	defer tracker.Stop()

	tracker.Start()
	testErr := errors.New("something broke")
	tracker.Fail(testErr)

	if tracker.State() != StateFailed {
		t.Errorf("expected failed, got %s", tracker.State())
	}
	if tracker.Context().LastErr == nil || tracker.Context().LastErr.Error() != "something broke" {
		t.Errorf("expected error to be recorded, got %v", tracker.Context().LastErr)
	}

	tracker.Reset()
	if tracker.State() != StatePending {
		t.Errorf("expected pending after reset, got %s", tracker.State())
	}
}

func TestTaskLifecycleRetry(t *testing.T) {
	tracker, err := NewTaskTracker("retry-task")
	if err != nil {
		t.Fatalf("NewTaskTracker: %v", err)
	}
	defer tracker.Stop()

	tracker.Start()
	if tracker.Context().Attempt != 1 {
		t.Errorf("expected attempt=1, got %d", tracker.Context().Attempt)
	}

	tracker.Retry()
	if tracker.State() != StateRetrying {
		t.Errorf("expected retrying, got %s", tracker.State())
	}

	tracker.Start()
	if tracker.State() != StateRunning {
		t.Errorf("expected running, got %s", tracker.State())
	}
	if tracker.Context().Attempt != 2 {
		t.Errorf("expected attempt=2, got %d", tracker.Context().Attempt)
	}

	tracker.Success()
	if !tracker.IsDone() {
		t.Error("expected done after success on retry")
	}
}

func TestTaskLifecycleAbort(t *testing.T) {
	tracker, err := NewTaskTracker("abort-task")
	if err != nil {
		t.Fatalf("NewTaskTracker: %v", err)
	}
	defer tracker.Stop()

	tracker.Start()
	tracker.Abort()

	if tracker.State() != StateAborted {
		t.Errorf("expected aborted, got %s", tracker.State())
	}
	if !tracker.IsDone() {
		t.Error("expected done after abort")
	}
}

func TestTaskLifecycleMatches(t *testing.T) {
	tracker, err := NewTaskTracker("matches-task")
	if err != nil {
		t.Fatalf("NewTaskTracker: %v", err)
	}
	defer tracker.Stop()

	if !tracker.Matches(StatePending) {
		t.Error("expected to match pending")
	}
	if tracker.Matches(StateRunning) {
		t.Error("should not match running")
	}
}
