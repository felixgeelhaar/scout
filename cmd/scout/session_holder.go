package main

import (
	"fmt"
	"sync"
	"time"

	"go.klarlabs.de/scout/agent"
)

// sessionHolder is the MCP process-wide session. get() returns a stable
// snapshot of the pointer while holding the mutex, so a concurrent configure
// cannot nil the package-level slot between ensure and use.
type sessionHolder struct {
	mu      sync.Mutex
	session *agent.Session
	cfg     agent.SessionConfig
	newFn   func(agent.SessionConfig) (*agent.Session, error)
}

func newSessionHolder(cfg agent.SessionConfig) *sessionHolder {
	return &sessionHolder{cfg: cfg, newFn: agent.NewSession}
}

func (h *sessionHolder) get() *agent.Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.session != nil {
		return h.session
	}
	s, err := h.newFn(h.cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create browser session: %v", err))
	}
	h.session = s
	return h.session
}

func (h *sessionHolder) config() agent.SessionConfig {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfg
}

func (h *sessionHolder) reconfigure(cfg agent.SessionConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg = cfg
	if h.session == nil {
		return
	}
	old := h.session
	h.session = nil
	go closeSessionSoon(old)
}

func (h *sessionHolder) close() {
	h.mu.Lock()
	s := h.session
	h.session = nil
	h.mu.Unlock()
	if s != nil {
		_ = s.Close()
	}
}

func closeSessionSoon(s *agent.Session) {
	done := make(chan struct{})
	go func() {
		defer func() {
			_ = recover()
			close(done)
		}()
		_ = s.Close()
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}
