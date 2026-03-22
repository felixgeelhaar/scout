package browse

import (
	"errors"
	"testing"
)

func TestContextNextCallsHandlersInOrder(t *testing.T) {
	var order []int

	handlers := HandlersChain{
		func(c *Context) { order = append(order, 1) },
		func(c *Context) { order = append(order, 2) },
		func(c *Context) { order = append(order, 3) },
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	if len(order) != 3 {
		t.Fatalf("expected 3 handlers called, got %d", len(order))
	}
	for i, v := range order {
		if v != i+1 {
			t.Errorf("handler %d: expected %d, got %d", i, i+1, v)
		}
	}
}

func TestContextAbortStopsChain(t *testing.T) {
	var order []int

	handlers := HandlersChain{
		func(c *Context) {
			order = append(order, 1)
		},
		func(c *Context) {
			order = append(order, 2)
			c.Abort()
		},
		func(c *Context) {
			order = append(order, 3) // should not be called
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	if len(order) != 2 {
		t.Fatalf("expected 2 handlers called, got %d: %v", len(order), order)
	}
	if !ctx.IsAborted() {
		t.Error("expected context to be aborted")
	}
}

func TestContextAbortWithError(t *testing.T) {
	testErr := errors.New("test error")

	handlers := HandlersChain{
		func(c *Context) {
			c.AbortWithError(testErr)
		},
		func(c *Context) {
			t.Error("should not be called")
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("expected context to be aborted")
	}
	errs := ctx.Errors()
	if len(errs) != 1 || errs[0] != testErr {
		t.Errorf("expected test error, got %v", errs)
	}
}

func TestContextMiddlewareNextFlow(t *testing.T) {
	var order []string

	handlers := HandlersChain{
		func(c *Context) {
			order = append(order, "before-1")
			c.Next()
			order = append(order, "after-1")
		},
		func(c *Context) {
			order = append(order, "before-2")
			c.Next()
			order = append(order, "after-2")
		},
		func(c *Context) {
			order = append(order, "handler")
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	expected := []string{"before-1", "before-2", "handler", "after-2", "after-1"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestContextSetGet(t *testing.T) {
	ctx := newContext(nil, "test", nil)

	ctx.Set("key1", "value1")
	ctx.Set("key2", 42)

	v1, ok := ctx.Get("key1")
	if !ok || v1 != "value1" {
		t.Errorf("Get key1: got %v, %v", v1, ok)
	}

	v2, ok := ctx.Get("key2")
	if !ok || v2 != 42 {
		t.Errorf("Get key2: got %v, %v", v2, ok)
	}

	_, ok = ctx.Get("missing")
	if ok {
		t.Error("Get missing: expected false")
	}

	s := ctx.GetString("key1")
	if s != "value1" {
		t.Errorf("GetString: got %q", s)
	}

	s = ctx.GetString("key2")
	if s != "" {
		t.Errorf("GetString non-string: expected empty, got %q", s)
	}
}

func TestContextTaskName(t *testing.T) {
	ctx := newContext(nil, "my-task", nil)
	if ctx.TaskName() != "my-task" {
		t.Errorf("expected 'my-task', got %q", ctx.TaskName())
	}
}

func TestContextSetGetBetweenMiddleware(t *testing.T) {
	handlers := HandlersChain{
		func(c *Context) {
			c.Set("user", "alice")
			c.Next()
		},
		func(c *Context) {
			user := c.GetString("user")
			if user != "alice" {
				panic("expected user=alice")
			}
			c.Set("result", "ok")
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	result := ctx.GetString("result")
	if result != "ok" {
		t.Errorf("expected result=ok, got %q", result)
	}
}

func TestContextSaveRestoreIndex(t *testing.T) {
	var attempts int

	handlers := HandlersChain{
		func(c *Context) {
			saved := c.SaveIndex()
			c.Next()
			if len(c.Errors()) > 0 && attempts < 2 {
				c.RestoreIndex(saved)
				c.Next()
			}
		},
		func(c *Context) {
			attempts++
			if attempts == 1 {
				c.AbortWithError(errors.New("first attempt failed"))
				return
			}
			c.Set("done", true)
		},
	}

	ctx := newContext(nil, "retry-test", handlers)
	ctx.Next()

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
	v, ok := ctx.Get("done")
	if !ok || v != true {
		t.Error("expected done=true after retry")
	}
}

func TestContextRestoreIndexClearsErrorsAndAborted(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	ctx.addError(errors.New("err1"))
	ctx.addError(errors.New("err2"))
	ctx.Abort()

	if !ctx.IsAborted() {
		t.Fatal("should be aborted")
	}
	if len(ctx.Errors()) != 2 {
		t.Fatalf("should have 2 errors, got %d", len(ctx.Errors()))
	}

	ctx.RestoreIndex(0)

	if ctx.IsAborted() {
		t.Error("RestoreIndex should clear aborted")
	}
	if len(ctx.Errors()) != 0 {
		t.Errorf("RestoreIndex should clear errors, got %d", len(ctx.Errors()))
	}
}

func TestContextRestoreIndexPreservesKeys(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	ctx.Set("preserved", "yes")
	ctx.addError(errors.New("some error"))
	ctx.Abort()

	ctx.RestoreIndex(0)

	v := ctx.GetString("preserved")
	if v != "yes" {
		t.Errorf("RestoreIndex should preserve keys, got %q", v)
	}
}

func TestContextGoContext(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	goCtx := ctx.GoContext()

	if goCtx == nil {
		t.Fatal("GoContext() returned nil")
	}
	if goCtx.Err() != nil {
		t.Error("GoContext should not be cancelled initially")
	}

	ctx.cancel()
	if goCtx.Err() == nil {
		t.Error("GoContext should be cancelled after context cancel")
	}
}

func TestContextPage(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	if ctx.Page() != nil {
		t.Error("Page() should be nil for context without page")
	}
}

func TestContextGetStringMissingKey(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	if s := ctx.GetString("nonexistent"); s != "" {
		t.Errorf("GetString for missing key should return empty, got %q", s)
	}
}

func TestContextGetStringNonStringValue(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	ctx.Set("num", 42)
	ctx.Set("bool", true)
	ctx.Set("nil", nil)

	if s := ctx.GetString("num"); s != "" {
		t.Errorf("GetString(int) should return empty, got %q", s)
	}
	if s := ctx.GetString("bool"); s != "" {
		t.Errorf("GetString(bool) should return empty, got %q", s)
	}
	if s := ctx.GetString("nil"); s != "" {
		t.Errorf("GetString(nil) should return empty, got %q", s)
	}
}

func TestContextEmptyHandlers(t *testing.T) {
	ctx := newContext(nil, "empty", HandlersChain{})
	ctx.Next()

	if ctx.IsAborted() {
		t.Error("empty handlers should not abort")
	}
	if len(ctx.Errors()) != 0 {
		t.Error("empty handlers should produce no errors")
	}
}

func TestContextNilHandlers(t *testing.T) {
	ctx := newContext(nil, "nil", nil)
	ctx.Next()

	if ctx.IsAborted() {
		t.Error("nil handlers should not abort")
	}
}

func TestContextMultipleErrors(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")
	err3 := errors.New("third")

	handlers := HandlersChain{
		func(c *Context) {
			c.addError(err1)
			c.addError(err2)
			c.addError(err3)
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()

	errs := ctx.Errors()
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(errs))
	}
	if errs[0] != err1 || errs[1] != err2 || errs[2] != err3 {
		t.Error("errors not in expected order")
	}
}

func TestNewTestContext(t *testing.T) {
	called := false
	handlers := HandlersChain{
		func(c *Context) {
			called = true
			c.Set("key", "val")
		},
	}

	ctx := NewTestContext("test-task", handlers)
	ctx.Next()

	if !called {
		t.Error("handler should have been called")
	}
	if ctx.TaskName() != "test-task" {
		t.Errorf("TaskName() = %q, want %q", ctx.TaskName(), "test-task")
	}
	if ctx.GetString("key") != "val" {
		t.Error("Set/GetString should work on test context")
	}
	if ctx.Page() != nil {
		t.Error("Page() should be nil on test context")
	}
}

func TestContextSaveIndex(t *testing.T) {
	handlers := HandlersChain{
		func(c *Context) {
			idx := c.SaveIndex()
			if idx != 0 {
				t.Errorf("SaveIndex at first handler = %d, want 0", idx)
			}
			c.Next()
		},
		func(c *Context) {
			idx := c.SaveIndex()
			if idx != 1 {
				t.Errorf("SaveIndex at second handler = %d, want 1", idx)
			}
		},
	}

	ctx := newContext(nil, "test", handlers)
	ctx.Next()
}

func TestContextAbortIdempotent(t *testing.T) {
	ctx := newContext(nil, "test", nil)

	ctx.Abort()
	if !ctx.IsAborted() {
		t.Error("should be aborted")
	}

	ctx.Abort()
	if !ctx.IsAborted() {
		t.Error("should still be aborted after double abort")
	}
}

func TestContextGetOverwrite(t *testing.T) {
	ctx := newContext(nil, "test", nil)
	ctx.Set("key", "first")
	ctx.Set("key", "second")

	v, ok := ctx.Get("key")
	if !ok || v != "second" {
		t.Errorf("Get after overwrite: got %v, %v", v, ok)
	}
}
