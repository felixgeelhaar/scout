package agent

import (
	"encoding/json"
	"testing"
)

func TestPageResult_JSON(t *testing.T) {
	tests := []struct {
		name string
		pr   PageResult
	}{
		{"basic", PageResult{URL: "https://example.com", Title: "Example"}},
		{"empty", PageResult{}},
		{"unicode", PageResult{URL: "https://example.com/日本語", Title: "日本語ページ"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.pr)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got PageResult
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got != tt.pr {
				t.Errorf("roundtrip: got %+v, want %+v", got, tt.pr)
			}
		})
	}
}

func TestElementResult_JSON(t *testing.T) {
	tests := []struct {
		name string
		er   ElementResult
	}{
		{"with text", ElementResult{Selector: "#foo", Text: "hello", Action: "extracted"}},
		{"with value", ElementResult{Selector: "input", Value: "bar", Action: "typed"}},
		{"omitempty text", ElementResult{Selector: "p", Action: "extracted"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.er)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got ElementResult
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got != tt.er {
				t.Errorf("roundtrip: got %+v, want %+v", got, tt.er)
			}
		})
	}
}

func TestExtractAllResult_JSON(t *testing.T) {
	r := ExtractAllResult{
		Selector:  "li",
		Count:     3,
		Total:     5,
		Truncated: true,
		Items:     []string{"a", "b", "c"},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got ExtractAllResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Selector != r.Selector || got.Count != r.Count || got.Total != r.Total || got.Truncated != r.Truncated {
		t.Errorf("roundtrip mismatch: got %+v", got)
	}
	if len(got.Items) != 3 {
		t.Errorf("items length: got %d, want 3", len(got.Items))
	}
}

func TestTableResult_JSON(t *testing.T) {
	r := TableResult{
		Selector:  "table",
		Headers:   []string{"Name", "Age"},
		Rows:      [][]string{{"Alice", "30"}, {"Bob", "25"}},
		RowCount:  2,
		ColCount:  2,
		Truncated: false,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got TableResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.RowCount != 2 || got.ColCount != 2 {
		t.Errorf("table counts: got %d/%d, want 2/2", got.RowCount, got.ColCount)
	}
	if len(got.Rows) != 2 || len(got.Headers) != 2 {
		t.Errorf("table sizes: rows=%d headers=%d", len(got.Rows), len(got.Headers))
	}
}

func TestFormResult_JSON(t *testing.T) {
	r := FormResult{
		Fields: []FieldResult{
			{Selector: "#email", Value: "test-user", Success: true},
			{Selector: "#missing", Error: "not found"},
		},
		Success: false,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got FormResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Success != false {
		t.Error("expected Success=false")
	}
	if len(got.Fields) != 2 {
		t.Fatalf("fields: got %d, want 2", len(got.Fields))
	}
	if got.Fields[0].Success != true || got.Fields[1].Error != "not found" {
		t.Errorf("field roundtrip mismatch")
	}
}

func TestObservation_JSON(t *testing.T) {
	obs := Observation{
		URL:         "https://example.com",
		Title:       "Example",
		Text:        "Hello World",
		Links:       []LinkInfo{{Text: "About", Href: "/about", Cost: "high"}},
		Inputs:      []InputInfo{{ID: "q", Type: "text", Placeholder: "Search"}},
		Buttons:     []ButtonInfo{{Text: "Submit", Type: "submit", Cost: "high"}},
		Interactive: 3,
		Meta:        map[string]string{"description": "test"},
		HasDialog:   true,
		DialogType:  "modal",
		DialogText:  "Confirm?",
	}
	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Observation
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.URL != obs.URL || got.Title != obs.Title || got.Interactive != 3 {
		t.Errorf("basic fields mismatch")
	}
	if !got.HasDialog || got.DialogType != "modal" || got.DialogText != "Confirm?" {
		t.Errorf("dialog fields mismatch")
	}
	if len(got.Links) != 1 || got.Links[0].Cost != "high" {
		t.Errorf("links mismatch")
	}
}

func TestDOMDiff_JSON(t *testing.T) {
	diff := DOMDiff{
		Added:          []DOMElement{{Tag: "div", ID: "new", Classes: "modal", Text: "Hello"}},
		Removed:        []DOMElement{{Tag: "span"}},
		Modified:       []DOMChange{{Tag: "input", Attribute: "class", OldValue: "a", NewValue: "b", ChangeType: "attribute"}},
		HasDiff:        true,
		Classification: "modal_appeared",
		Summary:        "Modal/dialog appeared: Hello",
	}
	data, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got DOMDiff
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !got.HasDiff || got.Classification != "modal_appeared" {
		t.Errorf("diff roundtrip mismatch")
	}
	if len(got.Added) != 1 || len(got.Removed) != 1 || len(got.Modified) != 1 {
		t.Errorf("diff element counts mismatch")
	}
}

func TestNetworkCapture_JSON(t *testing.T) {
	nc := NetworkCapture{
		URL:                   "https://api.example.com/data",
		Method:                "POST",
		Status:                201,
		MimeType:              "application/json",
		RequestHeaders:        map[string]string{"Content-Type": "application/json"},
		ResponseHeaders:       map[string]string{"X-Request-Id": "abc"},
		RequestBody:           `{"key":"value"}`,
		ResponseBody:          `{"id":1}`,
		RequestBodyTruncated:  false,
		ResponseBodyTruncated: true,
	}
	data, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got NetworkCapture
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.URL != nc.URL || got.Method != nc.Method || got.Status != nc.Status {
		t.Errorf("basic fields mismatch")
	}
	if !got.ResponseBodyTruncated || got.RequestBodyTruncated {
		t.Errorf("truncated flags mismatch")
	}
}

func TestFormFieldInfo_JSON(t *testing.T) {
	f := FormFieldInfo{
		Selector:    "#email",
		Label:       "Email Address",
		Type:        "email",
		Name:        "email",
		ID:          "email",
		Placeholder: "you-example",
		Required:    true,
		Options:     []string{"opt1", "opt2"},
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got FormFieldInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Label != f.Label || got.Required != true || len(got.Options) != 2 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestBatchAction_JSON(t *testing.T) {
	ba := BatchAction{
		Action:   "fill_form_semantic",
		Selector: "form",
		Value:    "test",
		Fields:   map[string]string{"Email": "ab-user"},
	}
	data, err := json.Marshal(ba)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got BatchAction
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Action != "fill_form_semantic" || got.Fields["Email"] != "ab-user" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestBatchResult_JSON(t *testing.T) {
	br := BatchResult{
		Total:     3,
		Succeeded: 2,
		Failed:    1,
		Results: []BatchActionResult{
			{Index: 0, Action: "click", Success: true},
			{Index: 1, Action: "type", Success: true},
			{Index: 2, Action: "wait", Success: false, Error: "timeout"},
		},
	}
	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got BatchResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Total != 3 || got.Succeeded != 2 || got.Failed != 1 {
		t.Errorf("counts mismatch")
	}
	if len(got.Results) != 3 || got.Results[2].Error != "timeout" {
		t.Errorf("results mismatch")
	}
}

func TestWebVitalsResult_JSON(t *testing.T) {
	wv := WebVitalsResult{
		LCP:              2500,
		CLS:              0.1,
		INP:              200,
		TTFB:             800,
		DOMContentLoaded: 1500,
		FirstPaint:       300,
		LCPRating:        "good",
		CLSRating:        "good",
		INPRating:        "good",
		OverallRating:    "good",
	}
	data, err := json.Marshal(wv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got WebVitalsResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.LCPRating != "good" || got.OverallRating != "good" {
		t.Errorf("ratings mismatch")
	}
	// Verify JSON field names
	m := make(map[string]any)
	json.Unmarshal(data, &m)
	if _, ok := m["lcp_ms"]; !ok {
		t.Error("expected JSON key 'lcp_ms'")
	}
	if _, ok := m["cls"]; !ok {
		t.Error("expected JSON key 'cls'")
	}
}

func TestAnnotatedElement_JSON(t *testing.T) {
	ae := AnnotatedElement{
		Label:    1,
		Selector: "#btn",
		Tag:      "button",
		Type:     "submit",
		Text:     "Go",
		Href:     "",
		X:        10,
		Y:        20,
		Width:    100,
		Height:   30,
	}
	data, err := json.Marshal(ae)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got AnnotatedElement
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Label != 1 || got.X != 10 || got.Width != 100 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestAnnotatedResult_ImageOmitted(t *testing.T) {
	ar := AnnotatedResult{
		Image:    []byte("png data"),
		Elements: []AnnotatedElement{{Label: 1, Tag: "a"}},
		Count:    1,
	}
	data, err := json.Marshal(ar)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	m := make(map[string]any)
	json.Unmarshal(data, &m)
	if _, ok := m["image"]; ok {
		t.Error("Image field should be omitted from JSON (json:\"-\")")
	}
}

func TestTraceEvent_JSON(t *testing.T) {
	te := TraceEvent{
		Index:     0,
		Action:    "navigate",
		URL:       "https://example.com",
		Timestamp: 1234567890,
		Duration:  500,
	}
	data, err := json.Marshal(te)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got TraceEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Action != "navigate" || got.Timestamp != 1234567890 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestPromptSelectResult_JSON(t *testing.T) {
	psr := PromptSelectResult{
		Selector:   "#login",
		Text:       "Log In",
		Tag:        "button",
		Role:       "button",
		Confidence: 0.95,
		Candidates: []PromptCandidate{
			{Selector: "#login", Text: "Log In", Score: 95},
			{Selector: "#signup", Text: "Sign Up", Score: 60},
		},
	}
	data, err := json.Marshal(psr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got PromptSelectResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Confidence != 0.95 || len(got.Candidates) != 2 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestHybridResult_JSON(t *testing.T) {
	hr := HybridResult{
		Elements: []HybridElement{
			{Index: 0, Tag: "button", Text: "Go", Selector: "#go", X: 10, Y: 20, Width: 50, Height: 30},
		},
		Width:  1920,
		Height: 1080,
	}
	data, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got HybridResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Width != 1920 || got.Height != 1080 || len(got.Elements) != 1 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestCookieDismissResult_JSON(t *testing.T) {
	cdr := CookieDismissResult{
		Found:    true,
		Method:   "selector",
		Selector: "#accept-cookies",
		Text:     "Accept All",
	}
	data, err := json.Marshal(cdr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got CookieDismissResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !got.Found || got.Method != "selector" || got.Text != "Accept All" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestPageReadiness_JSON(t *testing.T) {
	pr := PageReadiness{
		Score:         80,
		State:         "complete",
		PendingXHR:    0,
		PendingImages: 2,
		HasSkeleton:   false,
		HasSpinner:    true,
		Suggestions:   []string{"Wait for 2 images to load"},
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got PageReadiness
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Score != 80 || got.State != "complete" || !got.HasSpinner {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	if len(got.Suggestions) != 1 {
		t.Errorf("suggestions: got %d, want 1", len(got.Suggestions))
	}
}

func TestSemanticFillResult_JSON(t *testing.T) {
	sfr := SemanticFillResult{
		Fields: []SemanticFieldResult{
			{HumanName: "Email", Selector: "#email", Value: "ab-user", Success: true},
			{HumanName: "Phone", Error: "no matching field found"},
		},
		Success: false,
	}
	data, err := json.Marshal(sfr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got SemanticFillResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Success || len(got.Fields) != 2 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestSelectorSuggestion_JSON(t *testing.T) {
	ss := SelectorSuggestion{
		Selector: "#login-btn",
		Tag:      "button",
		Text:     "Log In",
		ID:       "login-btn",
		Classes:  "btn primary",
	}
	data, err := json.Marshal(ss)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got SelectorSuggestion
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Selector != "#login-btn" || got.Tag != "button" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}
