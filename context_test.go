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
