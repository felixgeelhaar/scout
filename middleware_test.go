package browse

import (
	"errors"
	"testing"
)

func TestRecoveryMiddleware(t *testing.T) {
	recovery := Recovery()

	handlers := HandlersChain{
		recovery,
		func(c *Context) {
			panic("test panic")
		},
	}

	ctx := newContext(nil, "panic-test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("expected context to be aborted after panic")
	}
	errs := ctx.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Error() != "test panic" {
		t.Errorf("expected 'test panic', got %q", errs[0].Error())
	}
}

func TestRecoveryMiddlewareWithErrorPanic(t *testing.T) {
	recovery := Recovery()

	handlers := HandlersChain{
		recovery,
		func(c *Context) {
			panic(&NavigationError{URL: "http://bad", Err: nil})
		},
	}

	ctx := newContext(nil, "panic-err-test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("expected context to be aborted")
	}
	errs := ctx.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestLoggerMiddleware(t *testing.T) {
	logger := Logger()

	called := false
	handlers := HandlersChain{
		logger,
		func(c *Context) {
			called = true
		},
	}

	ctx := newContext(nil, "log-test", handlers)
	ctx.Next()

	if !called {
		t.Error("handler was not called through logger middleware")
	}
}

func TestLoggerMiddlewareWithError(t *testing.T) {
	logger := Logger()

	handlers := HandlersChain{
		logger,
		func(c *Context) {
			c.AbortWithError(errors.New("something failed"))
		},
	}

	ctx := newContext(nil, "error-log-test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("context should be aborted")
	}
	if len(ctx.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(ctx.Errors()))
	}
}

func TestRecoveryWithNonErrorPanic(t *testing.T) {
	recovery := Recovery()

	handlers := HandlersChain{
		recovery,
		func(c *Context) {
			panic(42)
		},
	}

	ctx := newContext(nil, "int-panic-test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("expected context to be aborted")
	}
	errs := ctx.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Error() != "42" {
		t.Errorf("expected error message '42', got %q", errs[0].Error())
	}
}

func TestRecoveryWithStringPanic(t *testing.T) {
	recovery := Recovery()

	handlers := HandlersChain{
		recovery,
		func(c *Context) {
			panic("string panic")
		},
	}

	ctx := newContext(nil, "string-panic-test", handlers)
	ctx.Next()

	if !ctx.IsAborted() {
		t.Error("expected context to be aborted")
	}
	errs := ctx.Errors()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Error() != "string panic" {
		t.Errorf("expected 'string panic', got %q", errs[0].Error())
	}
}

func TestSetLogger(t *testing.T) {
	original := getLogger()
	if original == nil {
		t.Fatal("default logger should not be nil")
	}

	newLogger := getLogger()
	SetLogger(newLogger)

	if getLogger() != newLogger {
		t.Error("SetLogger should replace the logger")
	}

	SetLogger(original)
}

func TestRecoveryDoesNotAbortOnSuccess(t *testing.T) {
	recovery := Recovery()

	handlers := HandlersChain{
		recovery,
		func(c *Context) {
			c.Set("ok", true)
		},
	}

	ctx := newContext(nil, "ok-test", handlers)
	ctx.Next()

	if ctx.IsAborted() {
		t.Error("context should not be aborted on success")
	}
	v, _ := ctx.Get("ok")
	if v != true {
		t.Error("handler should have set ok=true")
	}
}
