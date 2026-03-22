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

func TestAttrMap_Empty(t *testing.T) {
	m := attrMap(nil)
	if len(m) != 0 {
		t.Errorf("attrMap(nil) should be empty, got %d entries", len(m))
	}

	m2 := attrMap([]string{})
	if len(m2) != 0 {
		t.Errorf("attrMap([]) should be empty, got %d entries", len(m2))
	}
}

func TestAttrMap_SinglePair(t *testing.T) {
	m := attrMap([]string{"href", "https://example.com"})
	if m["href"] != "https://example.com" {
		t.Errorf("attrMap[href] = %q", m["href"])
	}
}

func TestAttrMap_DuplicateKeys(t *testing.T) {
	m := attrMap([]string{"class", "first", "class", "second"})
	if m["class"] != "second" {
		t.Errorf("duplicate key should use last value, got %q", m["class"])
	}
}

func TestAttrMap_EmptyValues(t *testing.T) {
	m := attrMap([]string{"disabled", "", "data-x", ""})
	if m["disabled"] != "" {
		t.Errorf("empty value should be empty string, got %q", m["disabled"])
	}
	if _, ok := m["disabled"]; !ok {
		t.Error("key with empty value should still be present")
	}
}

func TestMatchFlatNode_EmptyNodes(t *testing.T) {
	got := matchFlatNode(nil, "#foo")
	if got != 0 {
		t.Errorf("matchFlatNode(nil, #foo) = %d, want 0", got)
	}

	got = matchFlatNode([]flatNode{}, ".bar")
	if got != 0 {
		t.Errorf("matchFlatNode([], .bar) = %d, want 0", got)
	}
}

func TestMatchFlatNode_EmptySelector(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV"},
	}
	got := matchFlatNode(nodes, "")
	if got != 0 {
		t.Errorf("matchFlatNode with empty selector = %d, want 0", got)
	}
}

func TestMatchFlatNode_ReturnsFirstMatch(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 10, NodeType: 1, NodeName: "DIV", Attributes: []string{"class", "item"}},
		{NodeID: 20, NodeType: 1, NodeName: "DIV", Attributes: []string{"class", "item"}},
	}
	got := matchFlatNode(nodes, ".item")
	if got != 10 {
		t.Errorf("matchFlatNode should return first match, got %d, want 10", got)
	}
}

func TestMatchFlatNode_TagCaseInsensitive(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "INPUT"},
	}
	tests := []struct {
		sel  string
		want int64
	}{
		{"input", 1},
		{"INPUT", 1},
		{"Input", 1},
		{"iNpUt", 1},
	}
	for _, tt := range tests {
		got := matchFlatNode(nodes, tt.sel)
		if got != tt.want {
			t.Errorf("matchFlatNode(%q) = %d, want %d", tt.sel, got, tt.want)
		}
	}
}

func TestMatchFlatNode_ClassPartialNoMatch(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV", Attributes: []string{"class", "button-primary"}},
	}
	got := matchFlatNode(nodes, ".button")
	if got != 0 {
		t.Errorf("partial class match should not match, got %d", got)
	}
}

func TestMatchFlatNode_IDExactMatch(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "DIV", Attributes: []string{"id", "main-content"}},
	}
	got := matchFlatNode(nodes, "#main")
	if got != 0 {
		t.Errorf("partial ID should not match, got %d", got)
	}
	got = matchFlatNode(nodes, "#main-content")
	if got != 1 {
		t.Errorf("exact ID should match, got %d", got)
	}
}

func TestMatchFlatNode_NoAttributes(t *testing.T) {
	nodes := []flatNode{
		{NodeID: 1, NodeType: 1, NodeName: "BR"},
	}
	got := matchFlatNode(nodes, "#anything")
	if got != 0 {
		t.Errorf("node without attrs should not match ID selector, got %d", got)
	}
	got = matchFlatNode(nodes, ".anything")
	if got != 0 {
		t.Errorf("node without attrs should not match class selector, got %d", got)
	}
	got = matchFlatNode(nodes, "br")
	if got != 1 {
		t.Errorf("node without attrs should match tag selector, got %d", got)
	}
}

func TestFlatNodeStruct(t *testing.T) {
	n := flatNode{
		NodeID:     42,
		NodeType:   1,
		NodeName:   "DIV",
		Attributes: []string{"id", "test", "class", "foo"},
	}
	if n.NodeID != 42 {
		t.Errorf("NodeID = %d, want 42", n.NodeID)
	}
	if n.NodeType != 1 {
		t.Errorf("NodeType = %d, want 1", n.NodeType)
	}
	if n.NodeName != "DIV" {
		t.Errorf("NodeName = %q, want DIV", n.NodeName)
	}
	if len(n.Attributes) != 4 {
		t.Errorf("Attributes len = %d, want 4", len(n.Attributes))
	}
}

func TestFlatNodeEmptyAttributes(t *testing.T) {
	n := flatNode{NodeID: 1, NodeType: 1, NodeName: "BR"}
	if n.Attributes != nil {
		t.Errorf("nil Attributes should be nil, got %v", n.Attributes)
	}
}
