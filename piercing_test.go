package browse

import "testing"

func TestMatchFlatNode_ID(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV", Attributes: []string{"id", "root"}},
		{NodeID: 2, NodeType: 1, NodeName: "SPAN", Attributes: []string{"id", "child", "class", "foo bar"}},
		{NodeID: 3, NodeType: 3, NodeName: "#text"},
	}

	tests := []struct {
		selector string
		want     int64
	}{
		{"#root", 1},
		{"#child", 2},
		{"#missing", 0},
	}

	for _, tt := range tests {
		got := matchFlatNode(nodes, tt.selector)
		if got != tt.want {
			t.Errorf("matchFlatNode(%q) = %d, want %d", tt.selector, got, tt.want)
		}
	}
}

func TestMatchFlatNode_Class(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV", Attributes: []string{"class", "alpha beta"}},
		{NodeID: 2, NodeType: 1, NodeName: "P", Attributes: []string{"class", "gamma"}},
	}

	tests := []struct {
		selector string
		want     int64
	}{
		{".alpha", 1},
		{".beta", 1},
		{".gamma", 2},
		{".missing", 0},
	}

	for _, tt := range tests {
		got := matchFlatNode(nodes, tt.selector)
		if got != tt.want {
			t.Errorf("matchFlatNode(%q) = %d, want %d", tt.selector, got, tt.want)
		}
	}
}

func TestMatchFlatNode_Tag(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV"},
		{NodeID: 2, NodeType: 1, NodeName: "SPAN"},
		{NodeID: 3, NodeType: 3, NodeName: "#text"},
	}

	tests := []struct {
		selector string
		want     int64
	}{
		{"div", 1},
		{"DIV", 1},
		{"span", 2},
		{"p", 0},
	}

	for _, tt := range tests {
		got := matchFlatNode(nodes, tt.selector)
		if got != tt.want {
			t.Errorf("matchFlatNode(%q) = %d, want %d", tt.selector, got, tt.want)
		}
	}
}

func TestMatchFlatNode_ComplexSelectorReturnsZero(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV", Attributes: []string{"id", "root"}},
	}
	selectors := []string{"div.foo", "div > span", "div + p", "div[data-x]", "div:first-child"}
	for _, sel := range selectors {
		got := matchFlatNode(nodes, sel)
		if got != 0 {
			t.Errorf("matchFlatNode(%q) = %d, want 0 for complex selector", sel, got)
		}
	}
}

func TestMatchFlatNode_SkipsNonElementNodes(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 3, NodeName: "#text", Attributes: []string{"id", "foo"}},
		{NodeID: 2, NodeType: 8, NodeName: "#comment"},
		{NodeID: 3, NodeType: 1, NodeName: "DIV", Attributes: []string{"id", "foo"}},
	}
	got := matchFlatNode(nodes, "#foo")
	if got != 3 {
		t.Errorf("matchFlatNode(#foo) = %d, want 3 (should skip text/comment nodes)", got)
	}
}

func TestAttrMap(t *testing.T) {
	attrs := []string{"id", "main", "class", "foo bar", "data-x", "123"}
	m := attrMap(attrs)
	if m["id"] != "main" {
		t.Errorf("attrMap[id] = %q, want %q", m["id"], "main")
	}
	if m["class"] != "foo bar" {
		t.Errorf("attrMap[class] = %q, want %q", m["class"], "foo bar")
	}
	if m["data-x"] != "123" {
		t.Errorf("attrMap[data-x] = %q, want %q", m["data-x"], "123")
	}
}

func TestAttrMap_OddLength(t *testing.T) {
	attrs := []string{"id", "main", "orphan"}
	m := attrMap(attrs)
	if m["id"] != "main" {
		t.Errorf("attrMap[id] = %q, want %q", m["id"], "main")
	}
	if _, ok := m["orphan"]; ok {
		t.Error("attrMap should not include orphan key with no value pair")
	}
}
