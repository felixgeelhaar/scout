package browse

import "encoding/json"

// Table represents extracted HTML table data.
type Table struct {
	Headers []string
	Rows    [][]string
}

// ExtractTable extracts data from an HTML table element.
// It reads <th> cells for headers and <td> cells for row data.
func (c *Context) ExtractTable(tableSelector string) (*Table, error) {
	// Extract headers
	headerSel := tableSelector + " th"
	headers := c.ElAll(headerSel)
	headerTexts, err := headers.Texts()
	if err != nil {
		headerTexts = nil
	}

	// Extract all rows and cells via a single JS evaluation
	js := `(function() {
		const table = document.querySelector(` + jsonQuote(tableSelector) + `);
		if (!table) return [];
		const rows = table.querySelectorAll('tbody tr, tr');
		const result = [];
		for (const row of rows) {
			const cells = row.querySelectorAll('td');
			if (cells.length === 0) continue;
			result.push(Array.from(cells).map(c => c.textContent.trim()));
		}
		return result;
	})()`

	rawResult, err := c.page.Evaluate(js)
	if err != nil {
		return &Table{Headers: headerTexts}, err
	}

	// rawResult is []any where each element is []any of strings
	var rows [][]string
	if arr, ok := rawResult.([]any); ok {
		for _, rowAny := range arr {
			if rowArr, ok := rowAny.([]any); ok {
				row := make([]string, 0, len(rowArr))
				for _, cellAny := range rowArr {
					s, _ := cellAny.(string)
					row = append(row, s)
				}
				rows = append(rows, row)
			}
		}
	}

	return &Table{Headers: headerTexts, Rows: rows}, nil
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
