package browse

import (
	"testing"
)

func TestGroupInheritsEngineMiddleware(t *testing.T) {
	e := New()

	var order []string
	e.Use(func(c *Context) {
		order = append(order, "engine")
		c.Next()
	})

	g := e.Group("auth")
	g.Use(func(c *Context) {
		order = append(order, "group")
		c.Next()
	})

	g.Task("login", func(c *Context) {
		order = append(order, "handler")
	})

	// Verify the chain was assembled correctly
	task := e.groups["auth"].tasks["auth/login"]
	if task == nil {
		t.Fatal("task 'auth/login' not found")
	}

	ctx := newContext(nil, task.name, task.handlers)
	ctx.Next()

	expected := []string{"engine", "group", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestSubGroup(t *testing.T) {
	e := New()

	var order []string
	e.Use(func(c *Context) {
		order = append(order, "engine")
		c.Next()
	})

	g := e.Group("admin")
	g.Use(func(c *Context) {
		order = append(order, "admin")
		c.Next()
	})

	sub := g.Group("users")
	sub.Use(func(c *Context) {
		order = append(order, "users")
		c.Next()
	})

	sub.Task("list", func(c *Context) {
		order = append(order, "handler")
	})

	task := e.groups["admin/users"].tasks["admin/users/list"]
	if task == nil {
		t.Fatal("task 'admin/users/list' not found")
	}

	ctx := newContext(nil, task.name, task.handlers)
	ctx.Next()

	expected := []string{"engine", "admin", "users", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestEngineTaskRegistration(t *testing.T) {
	e := New()
	e.Task("test", func(c *Context) {})

	if _, ok := e.tasks["test"]; !ok {
		t.Error("task 'test' not registered")
	}
}

func TestEngineFindTaskInGroup(t *testing.T) {
	e := New()
	g := e.Group("grp")
	g.Task("t1", func(c *Context) {})

	task, err := e.findTask("grp/t1")
	if err != nil {
		t.Fatalf("findTask: %v", err)
	}
	if task.name != "grp/t1" {
		t.Errorf("expected name 'grp/t1', got %q", task.name)
	}
}

func TestEngineFindTaskNotFound(t *testing.T) {
	e := New()
	_, err := e.findTask("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}
