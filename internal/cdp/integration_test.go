package cdp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockCDPServer creates a test server that echoes CDP responses.
// The handler function processes each incoming request and writes a response.
func mockCDPServer(t *testing.T, handler func(conn *websocket.Conn, req map[string]any)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer ws.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			var req map[string]any
			if err := json.Unmarshal(msg, &req); err != nil {
				continue
			}
			handler(ws, req)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// wsURL converts an httptest.Server URL to a ws:// URL.
func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// --- Conn.Call tests ---

func TestCallSuccessfulResult(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{
			"id":     id,
			"result": map[string]any{"frameId": "ABC123", "loaderId": "loader1"},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	result, err := conn.Call("Page.navigate", map[string]string{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	var resp struct {
		FrameID  string `json:"frameId"`
		LoaderID string `json:"loaderId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if resp.FrameID != "ABC123" {
		t.Errorf("frameId = %q, want %q", resp.FrameID, "ABC123")
	}
	if resp.LoaderID != "loader1" {
		t.Errorf("loaderId = %q, want %q", resp.LoaderID, "loader1")
	}
}

func TestCallErrorResponse(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{
			"id": id,
			"error": map[string]any{
				"code":    -32000,
				"message": "Cannot navigate to invalid URL",
			},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	_, err = conn.Call("Page.navigate", map[string]string{"url": "invalid"})
	if err == nil {
		t.Fatal("expected error from Call, got nil")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected *RPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32000 {
		t.Errorf("error code = %d, want -32000", rpcErr.Code)
	}
	if !strings.Contains(rpcErr.Message, "Cannot navigate") {
		t.Errorf("error message = %q, want contains 'Cannot navigate'", rpcErr.Message)
	}
}

func TestCallTimeout(t *testing.T) {
	// Server reads but never responds.
	srv := mockCDPServer(t, func(_ *websocket.Conn, _ map[string]any) {
		// intentionally no response
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = conn.CallSessionCtx(ctx, "", "Page.navigate", map[string]string{"url": "https://example.com"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("error = %q, want context deadline exceeded", err.Error())
	}
}

func TestCallNilParams(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		// Verify no params field was sent (or it's null).
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	result, err := conn.Call("Runtime.enable", nil)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// --- Conn.CallSessionCtx tests ---

func TestCallSessionCtxWithSessionID(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		sessionID, _ := req["sessionId"].(string)
		// Echo back the session ID in the result so we can verify it was sent.
		resp := map[string]any{
			"id":     id,
			"result": map[string]any{"echoSession": sessionID},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	result, err := conn.CallSessionCtx(ctx, "session-42", "DOM.getDocument", nil)
	if err != nil {
		t.Fatalf("CallSessionCtx error: %v", err)
	}

	var resp struct {
		EchoSession string `json:"echoSession"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if resp.EchoSession != "session-42" {
		t.Errorf("echoed session = %q, want %q", resp.EchoSession, "session-42")
	}
}

func TestCallSessionCtxCancellation(t *testing.T) {
	// Server delays response so we can cancel mid-flight.
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		time.Sleep(500 * time.Millisecond)
		id := req["id"]
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err = conn.CallSessionCtx(ctx, "s1", "Page.navigate", nil)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error = %q, want context canceled", err.Error())
	}
}

// --- Event dispatch over WebSocket ---

func TestEventDispatchOverWebSocket(t *testing.T) {
	// Server sends an event after receiving any call.
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		// First send the response.
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)

		// Then send an event.
		event := map[string]any{
			"method": "Page.loadEventFired",
			"params": map[string]any{"timestamp": 12345.678},
		}
		data, _ = json.Marshal(event)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	eventCh := make(chan json.RawMessage, 1)
	conn.On("Page.loadEventFired", func(params json.RawMessage) {
		eventCh <- params
	})

	// Trigger the server to send the event by making a call.
	_, err = conn.Call("Page.enable", nil)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	select {
	case params := <-eventCh:
		var p struct {
			Timestamp float64 `json:"timestamp"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			t.Fatalf("Unmarshal event params: %v", err)
		}
		if p.Timestamp != 12345.678 {
			t.Errorf("timestamp = %v, want 12345.678", p.Timestamp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSessionScopedEventOverWebSocket(t *testing.T) {
	// Server sends session-scoped events.
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)

		// Send event for session "s1".
		event := map[string]any{
			"sessionId": "s1",
			"method":    "DOM.documentUpdated",
			"params":    map[string]any{},
		}
		data, _ = json.Marshal(event)
		ws.WriteMessage(websocket.TextMessage, data)

		// Send event for session "s2".
		event["sessionId"] = "s2"
		data, _ = json.Marshal(event)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	var s1Count, s2Count atomic.Int32
	conn.OnSession("s1", "DOM.documentUpdated", func(_ json.RawMessage) {
		s1Count.Add(1)
	})
	conn.OnSession("s2", "DOM.documentUpdated", func(_ json.RawMessage) {
		s2Count.Add(1)
	})

	// Trigger events.
	_, err = conn.Call("DOM.enable", nil)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	// Wait for events to be dispatched.
	deadline := time.After(2 * time.Second)
	for {
		if s1Count.Load() >= 1 && s2Count.Load() >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out: s1=%d s2=%d", s1Count.Load(), s2Count.Load())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if s1Count.Load() != 1 {
		t.Errorf("s1 handler called %d times, want 1", s1Count.Load())
	}
	if s2Count.Load() != 1 {
		t.Errorf("s2 handler called %d times, want 1", s2Count.Load())
	}
}

func TestUnsubscribeOverWebSocket(t *testing.T) {
	var callCount atomic.Int32
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)

		// Send an event each time.
		event := map[string]any{
			"method": "Network.requestWillBeSent",
			"params": map[string]any{"requestId": callCount.Add(1)},
		}
		data, _ = json.Marshal(event)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	var handlerCalls atomic.Int32
	unsub := conn.On("Network.requestWillBeSent", func(_ json.RawMessage) {
		handlerCalls.Add(1)
	})

	// First call triggers event, handler should fire.
	_, _ = conn.Call("Network.enable", nil)
	time.Sleep(100 * time.Millisecond)
	if handlerCalls.Load() != 1 {
		t.Fatalf("handler called %d times after first call, want 1", handlerCalls.Load())
	}

	// Unsubscribe.
	unsub()

	// Second call triggers another event, handler should NOT fire.
	_, _ = conn.Call("Network.enable", nil)
	time.Sleep(100 * time.Millisecond)
	if handlerCalls.Load() != 1 {
		t.Errorf("handler called %d times after unsub, want still 1", handlerCalls.Load())
	}
}

func TestRemoveSessionHandlersOverWebSocket(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)

		// Send session-scoped event.
		event := map[string]any{
			"sessionId": "target-sess",
			"method":    "Page.frameNavigated",
			"params":    map[string]any{},
		}
		data, _ = json.Marshal(event)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	var calls atomic.Int32
	conn.OnSession("target-sess", "Page.frameNavigated", func(_ json.RawMessage) {
		calls.Add(1)
	})

	// First call -- handler fires.
	_, _ = conn.Call("Page.enable", nil)
	time.Sleep(100 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("handler called %d times, want 1", calls.Load())
	}

	// Remove all handlers for the session.
	conn.RemoveSessionHandlers("target-sess")

	// Second call -- handler should not fire.
	_, _ = conn.Call("Page.enable", nil)
	time.Sleep(100 * time.Millisecond)
	if calls.Load() != 1 {
		t.Errorf("handler called %d times after removal, want still 1", calls.Load())
	}
}

// --- Connection lifecycle ---

func TestDialAndClose(t *testing.T) {
	srv := mockCDPServer(t, func(_ *websocket.Conn, _ map[string]any) {})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}

	if err := conn.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}

	// Double close should be safe (idempotent via isClosed).
	if err := conn.Close(); err != nil {
		t.Errorf("second Close error: %v", err)
	}
}

func TestDialInvalidURL(t *testing.T) {
	_, err := Dial("ws://127.0.0.1:1") // port 1 should fail
	if err == nil {
		t.Fatal("expected Dial error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "cdp: failed to connect") {
		t.Errorf("error = %q, want contains 'cdp: failed to connect'", err.Error())
	}
}

func TestCloseInterruptsWaitingCall(t *testing.T) {
	// Server never responds.
	srv := mockCDPServer(t, func(_ *websocket.Conn, _ map[string]any) {})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := conn.CallSessionCtx(ctx, "", "Page.navigate", nil)
		errCh <- err
	}()

	// Give the call time to register, then close.
	time.Sleep(50 * time.Millisecond)
	conn.Close()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error after Close, got nil")
		}
		if !strings.Contains(err.Error(), "closed") {
			t.Errorf("error = %q, want contains 'closed'", err.Error())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for call to return after Close")
	}
}

func TestReadLoopClosesOnServerDisconnect(t *testing.T) {
	// Server accepts connection then immediately closes it.
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Close immediately to simulate server disconnect.
		ws.Close()
	}))
	defer srv.Close()

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	// The readLoop should detect the disconnect and close the connection.
	// Any call should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = conn.CallSessionCtx(ctx, "", "Page.enable", nil)
	if err == nil {
		t.Fatal("expected error after server disconnect, got nil")
	}
}

// --- Concurrent calls ---

func TestConcurrentCalls(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		method, _ := req["method"].(string)
		resp := map[string]any{
			"id":     id,
			"result": map[string]any{"method": method},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	const numCalls = 50
	var wg sync.WaitGroup
	errs := make(chan error, numCalls)

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := conn.Call("Test.method", map[string]int{"i": i})
			if err != nil {
				errs <- err
				return
			}
			var resp struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(result, &resp); err != nil {
				errs <- err
				return
			}
			if resp.Method != "Test.method" {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent call error: %v", err)
	}
}

// --- Target helper tests ---

func TestCreateTarget(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "Target.createTarget" {
			resp := map[string]any{
				"id":     id,
				"result": map[string]any{"targetId": "target-123"},
			}
			data, _ := json.Marshal(resp)
			ws.WriteMessage(websocket.TextMessage, data)
		}
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	targetID, err := conn.CreateTarget("about:blank")
	if err != nil {
		t.Fatalf("CreateTarget error: %v", err)
	}
	if targetID != "target-123" {
		t.Errorf("targetId = %q, want %q", targetID, "target-123")
	}
}

func TestAttachToTarget(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "Target.attachToTarget" {
			resp := map[string]any{
				"id":     id,
				"result": map[string]any{"sessionId": "session-abc"},
			}
			data, _ := json.Marshal(resp)
			ws.WriteMessage(websocket.TextMessage, data)
		}
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	sessionID, err := conn.AttachToTarget("target-123")
	if err != nil {
		t.Fatalf("AttachToTarget error: %v", err)
	}
	if sessionID != "session-abc" {
		t.Errorf("sessionId = %q, want %q", sessionID, "session-abc")
	}
}

func TestCloseTarget(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "Target.closeTarget" {
			resp := map[string]any{
				"id":     id,
				"result": map[string]any{"success": true},
			}
			data, _ := json.Marshal(resp)
			ws.WriteMessage(websocket.TextMessage, data)
		}
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	err = conn.CloseTarget("target-123")
	if err != nil {
		t.Errorf("CloseTarget error: %v", err)
	}
}

func TestCreateTargetError(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		resp := map[string]any{
			"id": id,
			"error": map[string]any{
				"code":    -32000,
				"message": "Target creation failed",
			},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	_, err = conn.CreateTarget("about:blank")
	if err == nil {
		t.Fatal("expected error from CreateTarget, got nil")
	}
}

// --- readLoop edge cases ---

func TestReadLoopInvalidJSON(t *testing.T) {
	// Server sends invalid JSON followed by a valid response.
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			// Send invalid JSON first -- readLoop should skip it.
			ws.WriteMessage(websocket.TextMessage, []byte("{not valid json"))

			// Then send a valid response.
			var req map[string]any
			json.Unmarshal(msg, &req)
			resp := map[string]any{
				"id":     req["id"],
				"result": map[string]any{"ok": true},
			}
			data, _ := json.Marshal(resp)
			ws.WriteMessage(websocket.TextMessage, data)
		}
	}))
	defer srv.Close()

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	// The call should succeed despite the invalid JSON in between.
	result, err := conn.Call("Test.method", nil)
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}

	var resp struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !resp.OK {
		t.Error("expected ok=true in response")
	}
}

func TestReadLoopClosesPendingOnDisconnect(t *testing.T) {
	// Server accepts, reads one message, then closes.
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Read one message then close without responding.
		ws.ReadMessage()
		ws.Close()
	}))
	defer srv.Close()

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = conn.CallSessionCtx(ctx, "", "Page.enable", nil)
	if err == nil {
		t.Fatal("expected error when server closes without responding, got nil")
	}
	// Should get "connection closed" not "deadline exceeded".
	if strings.Contains(err.Error(), "deadline") {
		t.Errorf("error = %q, expected connection closed not deadline", err.Error())
	}
}

// --- CallSession (non-ctx variant) ---

func TestCallSession(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		sessionID, _ := req["sessionId"].(string)
		resp := map[string]any{
			"id":     id,
			"result": map[string]any{"session": sessionID},
		}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	result, err := conn.CallSession("my-session", "DOM.enable", nil)
	if err != nil {
		t.Fatalf("CallSession error: %v", err)
	}

	var resp struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if resp.Session != "my-session" {
		t.Errorf("session = %q, want %q", resp.Session, "my-session")
	}
}

// --- Multiple events from single call ---

func TestMultipleEventsDispatched(t *testing.T) {
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := req["id"]
		// Respond first.
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)

		// Send 3 events.
		for i := 0; i < 3; i++ {
			event := map[string]any{
				"method": "Console.messageAdded",
				"params": map[string]any{"index": i},
			}
			data, _ = json.Marshal(event)
			ws.WriteMessage(websocket.TextMessage, data)
		}
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	var count atomic.Int32
	conn.On("Console.messageAdded", func(_ json.RawMessage) {
		count.Add(1)
	})

	_, _ = conn.Call("Console.enable", nil)

	// Wait for all 3 events.
	deadline := time.After(2 * time.Second)
	for count.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("timed out: received %d events, want 3", count.Load())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// --- ID monotonicity ---

func TestIDMonotonicallyIncreases(t *testing.T) {
	var lastID atomic.Int64
	srv := mockCDPServer(t, func(ws *websocket.Conn, req map[string]any) {
		id := int64(req["id"].(float64))
		prev := lastID.Swap(id)
		if prev >= id {
			t.Errorf("non-monotonic ID: prev=%d, got=%d", prev, id)
		}
		resp := map[string]any{"id": id, "result": map[string]any{}}
		data, _ := json.Marshal(resp)
		ws.WriteMessage(websocket.TextMessage, data)
	})

	conn, err := Dial(wsURL(srv))
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 10; i++ {
		_, err := conn.Call("Test.ping", nil)
		if err != nil {
			t.Fatalf("Call %d error: %v", i, err)
		}
	}
}
