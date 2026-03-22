package cdp

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
)

func TestMessageJSON(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
		want map[string]any
	}{
		{
			name: "method call with params",
			msg: Message{
				ID:     1,
				Method: "Page.navigate",
				Params: json.RawMessage(`{"url":"https://example.com"}`),
			},
			want: map[string]any{
				"id":     float64(1),
				"method": "Page.navigate",
			},
		},
		{
			name: "session-scoped call",
			msg: Message{
				ID:        2,
				SessionID: "sess-abc",
				Method:    "DOM.getDocument",
			},
			want: map[string]any{
				"id":        float64(2),
				"sessionId": "sess-abc",
				"method":    "DOM.getDocument",
			},
		},
		{
			name: "response with result",
			msg: Message{
				ID:     3,
				Result: json.RawMessage(`{"nodeId":1}`),
			},
			want: map[string]any{
				"id": float64(3),
			},
		},
		{
			name: "response with error",
			msg: Message{
				ID:    4,
				Error: &RPCError{Code: -32601, Message: "method not found"},
			},
			want: map[string]any{
				"id": float64(4),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("missing key %q in marshaled JSON", key)
					continue
				}
				if fmt.Sprintf("%v", gotVal) != fmt.Sprintf("%v", wantVal) {
					t.Errorf("key %q = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestMessageJSONOmitEmpty(t *testing.T) {
	// Verify that zero-value fields are omitted (omitempty behavior).
	msg := Message{}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	// All fields have omitempty, so an empty message should produce {}.
	if len(got) != 0 {
		t.Errorf("empty Message marshaled to %s, want {}", string(data))
	}
}

func TestMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "event message",
			json: `{"method":"Page.loadEventFired","params":{"timestamp":123.456},"sessionId":"s1"}`,
		},
		{
			name: "response message",
			json: `{"id":42,"result":{"frameId":"main"}}`,
		},
		{
			name: "error response",
			json: `{"id":5,"error":{"code":-32000,"message":"Not allowed"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			if err := json.Unmarshal([]byte(tt.json), &msg); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var msg2 Message
			if err := json.Unmarshal(data, &msg2); err != nil {
				t.Fatalf("second Unmarshal error: %v", err)
			}

			// Compare key fields.
			if msg.ID != msg2.ID {
				t.Errorf("ID mismatch: %d vs %d", msg.ID, msg2.ID)
			}
			if msg.Method != msg2.Method {
				t.Errorf("Method mismatch: %q vs %q", msg.Method, msg2.Method)
			}
			if msg.SessionID != msg2.SessionID {
				t.Errorf("SessionID mismatch: %q vs %q", msg.SessionID, msg2.SessionID)
			}
		})
	}
}

func TestRPCErrorInterface(t *testing.T) {
	tests := []struct {
		name    string
		err     RPCError
		wantMsg string
	}{
		{
			name:    "method not found",
			err:     RPCError{Code: -32601, Message: "method not found"},
			wantMsg: "cdp: error -32601: method not found",
		},
		{
			name:    "internal error",
			err:     RPCError{Code: -32603, Message: "internal error"},
			wantMsg: "cdp: error -32603: internal error",
		},
		{
			name:    "custom error",
			err:     RPCError{Code: -32000, Message: "Cannot navigate to invalid URL"},
			wantMsg: "cdp: error -32000: Cannot navigate to invalid URL",
		},
		{
			name:    "zero code",
			err:     RPCError{Code: 0, Message: "unknown"},
			wantMsg: "cdp: error 0: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("RPCError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestRPCErrorSatisfiesInterface(t *testing.T) {
	var err error = &RPCError{Code: -1, Message: "test"}
	if err.Error() == "" {
		t.Error("RPCError should produce non-empty error string")
	}
}

func TestEventKey(t *testing.T) {
	tests := []struct {
		name string
		a    eventKey
		b    eventKey
		same bool
	}{
		{
			name: "identical keys",
			a:    eventKey{sessionID: "s1", method: "Page.loaded"},
			b:    eventKey{sessionID: "s1", method: "Page.loaded"},
			same: true,
		},
		{
			name: "different session",
			a:    eventKey{sessionID: "s1", method: "Page.loaded"},
			b:    eventKey{sessionID: "s2", method: "Page.loaded"},
			same: false,
		},
		{
			name: "different method",
			a:    eventKey{sessionID: "s1", method: "Page.loaded"},
			b:    eventKey{sessionID: "s1", method: "Page.navigated"},
			same: false,
		},
		{
			name: "global event keys",
			a:    eventKey{sessionID: "", method: "Target.targetCreated"},
			b:    eventKey{sessionID: "", method: "Target.targetCreated"},
			same: true,
		},
		{
			name: "global vs session",
			a:    eventKey{sessionID: "", method: "Page.loaded"},
			b:    eventKey{sessionID: "s1", method: "Page.loaded"},
			same: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// eventKey is used as a map key; test equality via map lookup.
			m := map[eventKey]bool{tt.a: true}
			got := m[tt.b]
			if got != tt.same {
				t.Errorf("eventKey equality = %v, want %v", got, tt.same)
			}
		})
	}
}

func TestEventKeyAsMapKey(t *testing.T) {
	m := make(map[eventKey]int)
	k1 := eventKey{sessionID: "s1", method: "DOM.documentUpdated"}
	k2 := eventKey{sessionID: "s2", method: "DOM.documentUpdated"}
	k3 := eventKey{sessionID: "s1", method: "DOM.documentUpdated"}

	m[k1] = 1
	m[k2] = 2

	if m[k1] != 1 {
		t.Errorf("m[k1] = %d, want 1", m[k1])
	}
	if m[k2] != 2 {
		t.Errorf("m[k2] = %d, want 2", m[k2])
	}
	// k3 is identical to k1, should resolve to same entry.
	if m[k3] != 1 {
		t.Errorf("m[k3] = %d, want 1 (same as k1)", m[k3])
	}
}

func TestHandlerEntryCancellation(t *testing.T) {
	entry := &handlerEntry{
		fn: func(_ json.RawMessage) {},
	}

	if entry.cancelled.Load() {
		t.Error("new handlerEntry should not be cancelled")
	}

	entry.cancelled.Store(true)
	if !entry.cancelled.Load() {
		t.Error("handlerEntry should be cancelled after Store(true)")
	}
}

func TestNextIDGeneration(t *testing.T) {
	var counter atomic.Int64

	ids := make([]int64, 100)
	for i := range ids {
		ids[i] = counter.Add(1)
	}

	// Verify IDs are sequential and start at 1.
	for i, id := range ids {
		expected := int64(i + 1)
		if id != expected {
			t.Errorf("id[%d] = %d, want %d", i, id, expected)
		}
	}
}

func TestDefaultCallTimeout(t *testing.T) {
	if DefaultCallTimeout <= 0 {
		t.Errorf("DefaultCallTimeout = %v, want positive duration", DefaultCallTimeout)
	}
	if DefaultCallTimeout.Seconds() != 30 {
		t.Errorf("DefaultCallTimeout = %v, want 30s", DefaultCallTimeout)
	}
}

func TestMessageParamsMarshaling(t *testing.T) {
	// Verify that arbitrary params can be marshaled into a Message.
	params := map[string]any{
		"url":            "https://example.com",
		"transitionType": "typed",
	}

	b, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(params) error: %v", err)
	}

	msg := Message{
		ID:     1,
		Method: "Page.navigate",
		Params: b,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal(msg) error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// Verify params survived round-trip.
	var gotParams map[string]any
	if err := json.Unmarshal(decoded.Params, &gotParams); err != nil {
		t.Fatalf("json.Unmarshal(params) error: %v", err)
	}
	if gotParams["url"] != "https://example.com" {
		t.Errorf("params.url = %v, want https://example.com", gotParams["url"])
	}
}

func TestDispatchEvent(t *testing.T) {
	// Test dispatchEvent logic without a WebSocket connection.
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	var globalCalled, sessionCalled int

	// Register global handler.
	globalEntry := &handlerEntry{fn: func(_ json.RawMessage) { globalCalled++ }}
	c.events[eventKey{sessionID: "", method: "Page.loadEventFired"}] = []*handlerEntry{globalEntry}

	// Register session-scoped handler.
	sessEntry := &handlerEntry{fn: func(_ json.RawMessage) { sessionCalled++ }}
	c.events[eventKey{sessionID: "s1", method: "Page.loadEventFired"}] = []*handlerEntry{sessEntry}

	// Dispatch event from session s1.
	c.dispatchEvent("s1", "Page.loadEventFired", nil)
	if globalCalled != 1 {
		t.Errorf("global handler called %d times, want 1", globalCalled)
	}
	if sessionCalled != 1 {
		t.Errorf("session handler called %d times, want 1", sessionCalled)
	}

	// Dispatch event from session s2 — only global should fire.
	c.dispatchEvent("s2", "Page.loadEventFired", nil)
	if globalCalled != 2 {
		t.Errorf("global handler called %d times, want 2", globalCalled)
	}
	if sessionCalled != 1 {
		t.Errorf("session handler should not fire for s2, called %d times", sessionCalled)
	}

	// Dispatch global event (empty sessionID) — only global should fire.
	c.dispatchEvent("", "Page.loadEventFired", nil)
	if globalCalled != 3 {
		t.Errorf("global handler called %d times, want 3", globalCalled)
	}
	if sessionCalled != 1 {
		t.Errorf("session handler should not fire for empty session, called %d times", sessionCalled)
	}
}

func TestDispatchEventCancelled(t *testing.T) {
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	var called int
	entry := &handlerEntry{fn: func(_ json.RawMessage) { called++ }}
	c.events[eventKey{sessionID: "", method: "test"}] = []*handlerEntry{entry}

	// Fire once before cancellation.
	c.dispatchEvent("", "test", nil)
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}

	// Cancel and fire again.
	entry.cancelled.Store(true)
	c.dispatchEvent("", "test", nil)
	if called != 1 {
		t.Errorf("cancelled handler called %d times, want still 1", called)
	}
}

func TestRemoveSessionHandlers(t *testing.T) {
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	entry1 := &handlerEntry{fn: func(_ json.RawMessage) {}}
	entry2 := &handlerEntry{fn: func(_ json.RawMessage) {}}
	globalEntry := &handlerEntry{fn: func(_ json.RawMessage) {}}

	c.events[eventKey{sessionID: "s1", method: "Page.loaded"}] = []*handlerEntry{entry1}
	c.events[eventKey{sessionID: "s1", method: "DOM.updated"}] = []*handlerEntry{entry2}
	c.events[eventKey{sessionID: "", method: "Page.loaded"}] = []*handlerEntry{globalEntry}

	c.RemoveSessionHandlers("s1")

	if _, ok := c.events[eventKey{sessionID: "s1", method: "Page.loaded"}]; ok {
		t.Error("session handler should be removed after RemoveSessionHandlers")
	}
	if _, ok := c.events[eventKey{sessionID: "s1", method: "DOM.updated"}]; ok {
		t.Error("session handler should be removed after RemoveSessionHandlers")
	}
	if _, ok := c.events[eventKey{sessionID: "", method: "Page.loaded"}]; !ok {
		t.Error("global handler should survive RemoveSessionHandlers")
	}
}

func TestOnSessionUnsubscribe(t *testing.T) {
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	var called int
	unsub := c.OnSession("s1", "Page.loaded", func(_ json.RawMessage) { called++ })

	// Fire the event.
	c.dispatchEvent("s1", "Page.loaded", nil)
	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}

	// Unsubscribe and fire again.
	unsub()
	c.dispatchEvent("s1", "Page.loaded", nil)
	if called != 1 {
		t.Errorf("after unsubscribe, handler called %d times, want still 1", called)
	}
}

func TestOnGlobalHandler(t *testing.T) {
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	var called int
	unsub := c.On("Target.targetCreated", func(_ json.RawMessage) { called++ })

	c.dispatchEvent("", "Target.targetCreated", nil)
	c.dispatchEvent("any-session", "Target.targetCreated", nil)

	if called != 2 {
		t.Errorf("global handler called %d times, want 2", called)
	}

	unsub()
	c.dispatchEvent("", "Target.targetCreated", nil)
	if called != 2 {
		t.Errorf("after unsubscribe, handler called %d times, want still 2", called)
	}
}

func TestDispatchEventWithParams(t *testing.T) {
	c := &Conn{
		pending: make(map[int64]chan *Message),
		events:  make(map[eventKey][]*handlerEntry),
		closed:  make(chan struct{}),
	}

	var receivedParams json.RawMessage
	entry := &handlerEntry{fn: func(params json.RawMessage) { receivedParams = params }}
	c.events[eventKey{sessionID: "", method: "Page.frameNavigated"}] = []*handlerEntry{entry}

	params := json.RawMessage(`{"frame":{"id":"main","url":"https://example.com"}}`)
	c.dispatchEvent("", "Page.frameNavigated", params)

	if receivedParams == nil {
		t.Fatal("handler did not receive params")
	}

	var got map[string]any
	if err := json.Unmarshal(receivedParams, &got); err != nil {
		t.Fatalf("json.Unmarshal(params) error: %v", err)
	}

	frame, ok := got["frame"].(map[string]any)
	if !ok {
		t.Fatal("expected frame object in params")
	}
	if frame["url"] != "https://example.com" {
		t.Errorf("frame.url = %v, want https://example.com", frame["url"])
	}
}
