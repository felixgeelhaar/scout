// Package cdp provides a low-level Chrome DevTools Protocol client over WebSocket.
package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a CDP JSON-RPC message.
type Message struct {
	ID        int64           `json:"id,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *RPCError       `json:"error,omitempty"`
}

// RPCError represents a CDP error response.
type RPCError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("cdp: error %d: %s", e.Code, e.Message)
}

// eventKey uniquely identifies a session-scoped event handler.
type eventKey struct {
	sessionID string
	method    string
}

// handlerEntry holds a handler and its cancellation flag.
type handlerEntry struct {
	fn        func(json.RawMessage)
	cancelled atomic.Bool
}

// Conn is a WebSocket connection to a CDP target.
type Conn struct {
	ws        *websocket.Conn
	nextID    atomic.Int64
	pending   map[int64]chan *Message
	pendingMu sync.Mutex
	wsMu      sync.Mutex // separate lock for WebSocket writes
	events    map[eventKey][]*handlerEntry
	eventsMu  sync.RWMutex
	closed    chan struct{}
	isClosed  atomic.Bool
}

// DefaultCallTimeout is the maximum time to wait for a CDP response.
var DefaultCallTimeout = 30 * time.Second

// Dial connects to a CDP WebSocket endpoint.
func Dial(url string) (*Conn, error) {
	dialer := websocket.Dialer{
		ReadBufferSize:  1 << 20, // 1MB for large responses (screenshots)
		WriteBufferSize: 32 * 1024,
	}
	ws, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("cdp: failed to connect to %s: %w", url, err)
	}
	ws.SetReadLimit(64 * 1024 * 1024) // 64MB max message size

	c := &Conn{
		ws:      ws,
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// Call sends a CDP method call (browser-level, no session).
func (c *Conn) Call(method string, params any) (json.RawMessage, error) {
	return c.CallSession("", method, params)
}

// CallSession sends a CDP method call on a specific session.
// Uses DefaultCallTimeout. For context-aware calls, use CallSessionCtx.
func (c *Conn) CallSession(sessionID, method string, params any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCallTimeout)
	defer cancel()
	return c.CallSessionCtx(ctx, sessionID, method, params)
}

// CallSessionCtx sends a CDP method call with context-based cancellation.
func (c *Conn) CallSessionCtx(ctx context.Context, sessionID, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		rawParams = b
	}

	msg := Message{
		ID:        id,
		SessionID: sessionID,
		Method:    method,
		Params:    rawParams,
	}

	ch := make(chan *Message, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	c.wsMu.Lock()
	err = c.ws.WriteMessage(websocket.TextMessage, data)
	c.wsMu.Unlock()
	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("cdp: write error: %w", err)
	}

	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("cdp: connection closed while waiting for response")
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	case <-c.closed:
		return nil, fmt.Errorf("cdp: connection closed")
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("cdp: %w waiting for %s (id=%d)", ctx.Err(), method, id)
	}
}

// On registers a global event handler (no session filtering).
// Returns an unsubscribe function.
func (c *Conn) On(method string, handler func(json.RawMessage)) func() {
	return c.OnSession("", method, handler)
}

// OnSession registers an event handler scoped to a specific session.
// Events from other sessions are ignored. Returns an unsubscribe function.
func (c *Conn) OnSession(sessionID, method string, handler func(json.RawMessage)) func() {
	entry := &handlerEntry{fn: handler}
	key := eventKey{sessionID: sessionID, method: method}

	c.eventsMu.Lock()
	c.events[key] = append(c.events[key], entry)
	c.eventsMu.Unlock()

	return func() {
		entry.cancelled.Store(true)
	}
}

// RemoveSessionHandlers removes all event handlers for a given session.
func (c *Conn) RemoveSessionHandlers(sessionID string) {
	c.eventsMu.Lock()
	for key := range c.events {
		if key.sessionID == sessionID {
			delete(c.events, key)
		}
	}
	c.eventsMu.Unlock()
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	if !c.isClosed.CompareAndSwap(false, true) {
		return nil
	}
	close(c.closed)
	return c.ws.Close()
}

func (c *Conn) readLoop() {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			c.pendingMu.Lock()
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = make(map[int64]chan *Message)
			c.pendingMu.Unlock()

			if c.isClosed.CompareAndSwap(false, true) {
				close(c.closed)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		if msg.ID > 0 {
			c.pendingMu.Lock()
			ch, ok := c.pending[msg.ID]
			if ok {
				delete(c.pending, msg.ID)
			}
			c.pendingMu.Unlock()
			if ok {
				ch <- &msg
			}
		} else if msg.Method != "" {
			c.dispatchEvent(msg.SessionID, msg.Method, msg.Params)
		}
	}
}

func (c *Conn) dispatchEvent(sessionID, method string, params json.RawMessage) {
	c.eventsMu.RLock()
	// Collect handlers: session-scoped first, then global
	var handlers []*handlerEntry
	if sessionID != "" {
		handlers = append(handlers, c.events[eventKey{sessionID: sessionID, method: method}]...)
	}
	handlers = append(handlers, c.events[eventKey{sessionID: "", method: method}]...)
	c.eventsMu.RUnlock()

	for _, h := range handlers {
		if !h.cancelled.Load() {
			h.fn(params)
		}
	}
}
