package main

import (
	"sync"
	"sync/atomic"
	"testing"

	"go.klarlabs.de/scout/agent"
)

func TestSessionHolder_GetSnapshotsUnderLock(t *testing.T) {
	var calls atomic.Int32
	sentinel := &agent.Session{}
	h := &sessionHolder{
		newFn: func(agent.SessionConfig) (*agent.Session, error) {
			calls.Add(1)
			return sentinel, nil
		},
	}

	const n = 32
	got := make([]*agent.Session, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			got[i] = h.get()
		}(i)
	}
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("newFn called %d times, want 1", calls.Load())
	}
	for i, s := range got {
		if s != sentinel {
			t.Errorf("got[%d] is not the snapshot pointer", i)
		}
	}
}

func TestSessionHolder_ReconfigureNilsSession(t *testing.T) {
	sentinel := &agent.Session{}
	h := &sessionHolder{
		newFn: func(agent.SessionConfig) (*agent.Session, error) {
			return sentinel, nil
		},
	}
	if h.get() != sentinel {
		t.Fatal("expected sentinel")
	}
	h.reconfigure(agent.SessionConfig{Headless: false})
	if h.session != nil {
		t.Fatal("reconfigure must nil the current session")
	}
	if h.config().Headless {
		t.Fatal("config not updated")
	}
}
