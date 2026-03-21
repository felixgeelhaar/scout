package browse

import "sync"

// Group represents a named collection of tasks with shared middleware.
type Group struct {
	name       string
	engine     *Engine
	middleware HandlersChain
	tasks      map[string]*taskEntry
	mu         sync.RWMutex
}

// Use appends middleware to this group. Group middleware runs after engine middleware.
func (g *Group) Use(middleware ...HandlerFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Task registers a named task within this group.
// The final handler chain is: engine middleware + group middleware + task handlers.
func (g *Group) Task(name string, handlers ...HandlerFunc) {
	chain := make(HandlersChain, 0, len(g.engine.middleware)+len(g.middleware)+len(handlers))
	chain = append(chain, g.engine.middleware...)
	chain = append(chain, g.middleware...)
	chain = append(chain, handlers...)

	fullName := g.name + "/" + name

	g.mu.Lock()
	g.tasks[fullName] = &taskEntry{name: fullName, handlers: chain}
	g.mu.Unlock()
}

// Group creates a sub-group that inherits this group's middleware.
func (g *Group) Group(name string, middleware ...HandlerFunc) *Group {
	combined := make(HandlersChain, 0, len(g.middleware)+len(middleware))
	combined = append(combined, g.middleware...)
	combined = append(combined, middleware...)

	sub := &Group{
		name:       g.name + "/" + name,
		engine:     g.engine,
		middleware: combined,
		tasks:      make(map[string]*taskEntry),
	}

	g.engine.mu.Lock()
	g.engine.groups[sub.name] = sub
	g.engine.mu.Unlock()

	return sub
}
