package browse

import "testing"

func TestTableStruct(t *testing.T) {
	table := &Table{
		Headers: []string{"Name", "Age", "City"},
		Rows: [][]string{
			{"Alice", "30", "NYC"},
			{"Bob", "25", "LA"},
		},
	}

	if len(table.Headers) != 3 {
		t.Errorf("Headers len = %d, want 3", len(table.Headers))
	}
	if table.Headers[0] != "Name" {
		t.Errorf("Headers[0] = %q, want %q", table.Headers[0], "Name")
	}
	if len(table.Rows) != 2 {
		t.Errorf("Rows len = %d, want 2", len(table.Rows))
	}
	if table.Rows[0][0] != "Alice" {
		t.Errorf("Rows[0][0] = %q, want %q", table.Rows[0][0], "Alice")
	}
}

func TestTableEmpty(t *testing.T) {
	table := &Table{}
	if table.Headers != nil {
		t.Error("empty table Headers should be nil")
	}
	if table.Rows != nil {
		t.Error("empty table Rows should be nil")
	}
}

func TestTableNoHeaders(t *testing.T) {
	table := &Table{
		Rows: [][]string{
			{"a", "b"},
			{"c", "d"},
		},
	}
	if table.Headers != nil {
		t.Error("table with no headers should have nil Headers")
	}
	if len(table.Rows) != 2 {
		t.Errorf("Rows len = %d, want 2", len(table.Rows))
	}
}

func TestTableNoRows(t *testing.T) {
	table := &Table{
		Headers: []string{"Col1", "Col2"},
	}
	if len(table.Headers) != 2 {
		t.Errorf("Headers len = %d, want 2", len(table.Headers))
	}
	if table.Rows != nil {
		t.Error("table with no rows should have nil Rows")
	}
}

func TestJsonQuoteForTable(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"css selector", "table.data", `"table.data"`},
		{"id selector", "#my-table", `"#my-table"`},
		{"with quotes", `table[data-name="x"]`, `"table[data-name=\"x\"]"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonQuote(tt.in)
			if got != tt.want {
				t.Errorf("jsonQuote(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
