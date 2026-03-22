package browse

import (
	"errors"
	"testing"
)

func TestSelectionAllCount(t *testing.T) {
	tests := []struct {
		name  string
		sels  []*Selection
		count int
	}{
		{"empty", nil, 0},
		{"one", []*Selection{{nodeID: 1}}, 1},
		{"three", []*Selection{{nodeID: 1}, {nodeID: 2}, {nodeID: 3}}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &SelectionAll{selections: tt.sels, selector: "div"}
			if sa.Count() != tt.count {
				t.Errorf("Count() = %d, want %d", sa.Count(), tt.count)
			}
		})
	}
}

func TestSelectionAllFirst(t *testing.T) {
	t.Run("with elements", func(t *testing.T) {
		sa := &SelectionAll{
			selections: []*Selection{
				{nodeID: 10, selector: "div"},
				{nodeID: 20, selector: "div"},
			},
			selector: "div",
		}
		first := sa.First()
		if first.nodeID != 10 {
			t.Errorf("First().nodeID = %d, want 10", first.nodeID)
		}
		if first.Err() != nil {
			t.Errorf("First().Err() = %v, want nil", first.Err())
		}
	})

	t.Run("empty returns error selection", func(t *testing.T) {
		sa := &SelectionAll{selections: nil, selector: ".missing"}
		first := sa.First()
		if first.Err() == nil {
			t.Error("First() on empty should return error")
		}
		var notFound *ElementNotFoundError
		if !errors.As(first.Err(), &notFound) {
			t.Errorf("First() error should be ElementNotFoundError, got %T", first.Err())
		}
	})

	t.Run("with error returns error selection", func(t *testing.T) {
		sa := &SelectionAll{err: errors.New("query failed"), selector: ".broken"}
		first := sa.First()
		if first.Err() == nil {
			t.Error("First() on errored SelectionAll should return error")
		}
	})
}

func TestSelectionAllLast(t *testing.T) {
	t.Run("with elements", func(t *testing.T) {
		sa := &SelectionAll{
			selections: []*Selection{
				{nodeID: 10},
				{nodeID: 20},
				{nodeID: 30},
			},
			selector: "span",
		}
		last := sa.Last()
		if last.nodeID != 30 {
			t.Errorf("Last().nodeID = %d, want 30", last.nodeID)
		}
	})

	t.Run("empty returns error selection", func(t *testing.T) {
		sa := &SelectionAll{selections: nil, selector: ".missing"}
		last := sa.Last()
		if last.Err() == nil {
			t.Error("Last() on empty should return error")
		}
	})

	t.Run("single element", func(t *testing.T) {
		sa := &SelectionAll{
			selections: []*Selection{{nodeID: 42}},
			selector:   "p",
		}
		last := sa.Last()
		if last.nodeID != 42 {
			t.Errorf("Last().nodeID = %d, want 42", last.nodeID)
		}
	})
}

func TestSelectionAllAt(t *testing.T) {
	sels := []*Selection{
		{nodeID: 10},
		{nodeID: 20},
		{nodeID: 30},
	}
	sa := &SelectionAll{selections: sels, selector: "li"}

	tests := []struct {
		name    string
		index   int
		wantID  int64
		wantErr bool
	}{
		{"first", 0, 10, false},
		{"middle", 1, 20, false},
		{"last", 2, 30, false},
		{"negative", -1, 0, true},
		{"out of bounds", 3, 0, true},
		{"way out", 100, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sa.At(tt.index)
			if tt.wantErr {
				if got.Err() == nil {
					t.Error("At() should return error for invalid index")
				}
			} else {
				if got.nodeID != tt.wantID {
					t.Errorf("At(%d).nodeID = %d, want %d", tt.index, got.nodeID, tt.wantID)
				}
			}
		})
	}

	t.Run("with error", func(t *testing.T) {
		errSA := &SelectionAll{err: errors.New("broken"), selector: "div"}
		got := errSA.At(0)
		if got.Err() == nil {
			t.Error("At() on errored SelectionAll should return error")
		}
	})
}

func TestSelectionAllEach(t *testing.T) {
	t.Run("iterates all", func(t *testing.T) {
		sels := []*Selection{
			{nodeID: 1},
			{nodeID: 2},
			{nodeID: 3},
		}
		sa := &SelectionAll{selections: sels, selector: "div"}

		var visited []int64
		err := sa.Each(func(i int, s *Selection) {
			visited = append(visited, s.nodeID)
		})
		if err != nil {
			t.Errorf("Each() returned error: %v", err)
		}
		if len(visited) != 3 {
			t.Fatalf("Each visited %d elements, want 3", len(visited))
		}
		for i, id := range []int64{1, 2, 3} {
			if visited[i] != id {
				t.Errorf("visited[%d] = %d, want %d", i, visited[i], id)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		sa := &SelectionAll{selections: nil, selector: "div"}
		called := false
		err := sa.Each(func(i int, s *Selection) {
			called = true
		})
		if err != nil {
			t.Errorf("Each() on empty returned error: %v", err)
		}
		if called {
			t.Error("Each() on empty should not call fn")
		}
	})

	t.Run("with error", func(t *testing.T) {
		testErr := errors.New("query failed")
		sa := &SelectionAll{err: testErr, selector: "div"}
		err := sa.Each(func(i int, s *Selection) {
			t.Error("should not be called")
		})
		if err != testErr {
			t.Errorf("Each() should return underlying error, got %v", err)
		}
	})
}

func TestSelectionAllFilter(t *testing.T) {
	t.Run("filters elements", func(t *testing.T) {
		sels := []*Selection{
			{nodeID: 1},
			{nodeID: 2},
			{nodeID: 3},
			{nodeID: 4},
		}
		sa := &SelectionAll{selections: sels, selector: "div"}

		filtered := sa.Filter(func(s *Selection) bool {
			return s.nodeID%2 == 0
		})

		if filtered.Count() != 2 {
			t.Fatalf("Filter count = %d, want 2", filtered.Count())
		}
		if filtered.At(0).nodeID != 2 {
			t.Errorf("filtered[0].nodeID = %d, want 2", filtered.At(0).nodeID)
		}
		if filtered.At(1).nodeID != 4 {
			t.Errorf("filtered[1].nodeID = %d, want 4", filtered.At(1).nodeID)
		}
	})

	t.Run("filter none match", func(t *testing.T) {
		sels := []*Selection{{nodeID: 1}, {nodeID: 2}}
		sa := &SelectionAll{selections: sels, selector: "div"}

		filtered := sa.Filter(func(s *Selection) bool { return false })
		if filtered.Count() != 0 {
			t.Errorf("Filter should return 0, got %d", filtered.Count())
		}
	})

	t.Run("filter all match", func(t *testing.T) {
		sels := []*Selection{{nodeID: 1}, {nodeID: 2}}
		sa := &SelectionAll{selections: sels, selector: "div"}

		filtered := sa.Filter(func(s *Selection) bool { return true })
		if filtered.Count() != 2 {
			t.Errorf("Filter should return 2, got %d", filtered.Count())
		}
	})

	t.Run("with error returns self", func(t *testing.T) {
		sa := &SelectionAll{err: errors.New("broken"), selector: "div"}
		filtered := sa.Filter(func(s *Selection) bool { return true })
		if filtered != sa {
			t.Error("Filter on errored SelectionAll should return self")
		}
	})

	t.Run("preserves selector", func(t *testing.T) {
		sels := []*Selection{{nodeID: 1}}
		sa := &SelectionAll{selections: sels, selector: ".item"}

		filtered := sa.Filter(func(s *Selection) bool { return true })
		if filtered.selector != ".item" {
			t.Errorf("Filter should preserve selector, got %q", filtered.selector)
		}
	})
}
