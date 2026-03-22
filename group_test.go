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

func TestEngineUseAppendsMiddleware(t *testing.T) {
	e := New()
	if len(e.middleware) != 0 {
		t.Fatalf("initial middleware len = %d, want 0", len(e.middleware))
	}

	e.Use(func(c *Context) {})
	if len(e.middleware) != 1 {
		t.Errorf("middleware len after Use = %d, want 1", len(e.middleware))
	}

	e.Use(func(c *Context) {}, func(c *Context) {})
	if len(e.middleware) != 3 {
		t.Errorf("middleware len after second Use = %d, want 3", len(e.middleware))
	}
}

func TestEngineCollectAllTasks(t *testing.T) {
	e := New()
	e.Task("root1", func(c *Context) {})
	e.Task("root2", func(c *Context) {})

	g := e.Group("grp")
	g.Task("t1", func(c *Context) {})
	g.Task("t2", func(c *Context) {})

	all := e.collectAllTasks()
	if len(all) != 4 {
		t.Errorf("collectAllTasks() = %d tasks, want 4", len(all))
	}
}

func TestEngineCollectAllTasksEmpty(t *testing.T) {
	e := New()
	all := e.collectAllTasks()
	if len(all) != 0 {
		t.Errorf("collectAllTasks() on empty = %d, want 0", len(all))
	}
}

func TestEngineTaskChainIncludesMiddleware(t *testing.T) {
	e := New()
	e.Use(func(c *Context) { c.Next() })
	e.Use(func(c *Context) { c.Next() })

	e.Task("test", func(c *Context) {})

	task := e.tasks["test"]
	if task == nil {
		t.Fatal("task not found")
	}
	if len(task.handlers) != 3 {
		t.Errorf("task chain len = %d, want 3 (2 middleware + 1 handler)", len(task.handlers))
	}
}

func TestEngineGroupCreation(t *testing.T) {
	e := New()
	g := e.Group("api")

	if g.name != "api" {
		t.Errorf("group name = %q, want %q", g.name, "api")
	}
	if g.engine != e {
		t.Error("group engine should reference parent engine")
	}
	if _, ok := e.groups["api"]; !ok {
		t.Error("group should be registered on engine")
	}
}

func TestEngineRunGroupNotFound(t *testing.T) {
	e := New()
	err := e.RunGroup("nonexistent")
	if err == nil {
		t.Error("RunGroup should return error for missing group")
	}
}

func TestGroupTaskNaming(t *testing.T) {
	e := New()
	g := e.Group("auth")
	g.Task("login", func(c *Context) {})

	if _, ok := g.tasks["auth/login"]; !ok {
		t.Error("task should be registered as 'auth/login'")
	}
}

func TestSubGroupTaskNaming(t *testing.T) {
	e := New()
	g := e.Group("api")
	sub := g.Group("v2")
	sub.Task("users", func(c *Context) {})

	if sub.name != "api/v2" {
		t.Errorf("sub-group name = %q, want %q", sub.name, "api/v2")
	}
	if _, ok := sub.tasks["api/v2/users"]; !ok {
		t.Error("task should be registered as 'api/v2/users'")
	}
}

func TestGroupMiddlewareChaining(t *testing.T) {
	e := New()
	e.Use(func(c *Context) { c.Next() })

	g := e.Group("grp", func(c *Context) { c.Next() })
	g.Use(func(c *Context) { c.Next() })

	g.Task("task", func(c *Context) {})

	task := g.tasks["grp/task"]
	if task == nil {
		t.Fatal("task not found")
	}
	if len(task.handlers) != 4 {
		t.Errorf("chain len = %d, want 4 (1 engine + 1 group ctor + 1 group use + 1 handler)", len(task.handlers))
	}
}

func TestSubGroupInheritsParentMiddleware(t *testing.T) {
	e := New()
	g := e.Group("parent", func(c *Context) { c.Next() })
	sub := g.Group("child", func(c *Context) { c.Next() })

	if len(sub.middleware) != 2 {
		t.Errorf("sub-group middleware len = %d, want 2 (parent + child)", len(sub.middleware))
	}
}

func TestEngineFindTaskInRootFirst(t *testing.T) {
	e := New()
	e.Task("shared", func(c *Context) {})

	task, err := e.findTask("shared")
	if err != nil {
		t.Fatalf("findTask: %v", err)
	}
	if task.name != "shared" {
		t.Errorf("task name = %q, want %q", task.name, "shared")
	}
}

func TestEngineLaunchAlreadyLaunchedWithoutConn(t *testing.T) {
	e := New()
	err := e.Launch()
	if err == nil {
		defer e.Close()
		return
	}
}

func TestEngineCloseWithoutLaunch(t *testing.T) {
	e := New()
	err := e.Close()
	if err != nil {
		t.Errorf("Close() without Launch should not error, got: %v", err)
	}
}

func TestEngineNewWithNoOptions(t *testing.T) {
	e := New()
	if e.opts.headless != true {
		t.Error("default headless should be true")
	}
	if e.opts.timeout != 30*1000*1000*1000 {
		t.Errorf("default timeout should be 30s, got %v", e.opts.timeout)
	}
	if e.groups == nil {
		t.Error("groups map should be initialized")
	}
	if e.tasks == nil {
		t.Error("tasks map should be initialized")
	}
}

func TestDefaultEngineHasLoggerAndRecovery(t *testing.T) {
	e := Default()
	if len(e.middleware) != 2 {
		t.Errorf("Default() middleware count = %d, want 2", len(e.middleware))
	}
}

func TestDefaultEngineWithOptions(t *testing.T) {
	e := Default(WithHeadless(false), WithViewport(800, 600))
	if e.opts.headless {
		t.Error("headless should be false")
	}
	if e.opts.width != 800 || e.opts.height != 600 {
		t.Errorf("viewport = %dx%d, want 800x600", e.opts.width, e.opts.height)
	}
	if len(e.middleware) != 2 {
		t.Errorf("middleware count = %d, want 2", len(e.middleware))
	}
}
