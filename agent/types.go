package agent

// PageResult is the structured response after a navigation or page-level action.
type PageResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ElementResult is the structured response for single-element operations.
type ElementResult struct {
	Selector string `json:"selector"`
	Text     string `json:"text,omitempty"`
	Value    string `json:"value,omitempty"`
	Action   string `json:"action"`
}

// ExtractAllResult is the structured response for multi-element extraction.
type ExtractAllResult struct {
	Selector  string   `json:"selector"`
	Count     int      `json:"count"`
	Total     int      `json:"total"`
	Truncated bool     `json:"truncated,omitempty"`
	Items     []string `json:"items"`
}

// TableResult is the structured response for table extraction.
type TableResult struct {
	Selector  string     `json:"selector"`
	Headers   []string   `json:"headers"`
	Rows      [][]string `json:"rows"`
	RowCount  int        `json:"row_count"`
	ColCount  int        `json:"col_count"`
	Truncated bool       `json:"truncated,omitempty"`
}

// FormResult is the structured response for form filling.
type FormResult struct {
	Fields  []FieldResult `json:"fields"`
	Success bool          `json:"success"`
}

// FieldResult describes the outcome of filling a single field.
type FieldResult struct {
	Selector string `json:"selector"`
	Value    string `json:"value,omitempty"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// Observation is a structured snapshot of the visible page for agent context.
type Observation struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Text        string            `json:"text"`
	Links       []LinkInfo        `json:"links,omitempty"`
	Inputs      []InputInfo       `json:"inputs,omitempty"`
	Buttons     []ButtonInfo      `json:"buttons,omitempty"`
	Interactive int               `json:"interactive_elements"`
	Meta        map[string]string `json:"meta,omitempty"`
}

// LinkInfo describes a link on the page.
type LinkInfo struct {
	Text string `json:"text"`
	Href string `json:"href"`
	Cost string `json:"cost,omitempty"` // "high" (navigation), "medium" (ajax), "low" (anchor)
}

// InputInfo describes an input element.
type InputInfo struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type"`
	Value       string `json:"value,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// ButtonInfo describes a button element.
type ButtonInfo struct {
	Text string `json:"text"`
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Cost string `json:"cost,omitempty"` // "high" (submit/navigation), "medium" (action), "low" (toggle)
}

// --- DOM Diff types ---

// DOMDiff represents changes between two Observe() calls.
type DOMDiff struct {
	Added          []DOMElement `json:"added,omitempty"`
	Removed        []DOMElement `json:"removed,omitempty"`
	Modified       []DOMChange  `json:"modified,omitempty"`
	HasDiff        bool         `json:"has_diff"`
	Classification string       `json:"classification,omitempty"` // navigation, content_loaded, modal_appeared, form_error, notification, loading_complete, element_state_changed, minor_update
	Summary        string       `json:"summary,omitempty"`        // human-readable one-line summary
}

// DOMElement describes an element that was added or removed.
type DOMElement struct {
	Tag     string `json:"tag"`
	ID      string `json:"id,omitempty"`
	Classes string `json:"classes,omitempty"`
	Text    string `json:"text,omitempty"`
}

// DOMChange describes a modification to an existing element.
type DOMChange struct {
	Tag        string `json:"tag"`
	ID         string `json:"id,omitempty"`
	Attribute  string `json:"attribute,omitempty"`
	OldValue   string `json:"old_value,omitempty"`
	NewValue   string `json:"new_value,omitempty"`
	ChangeType string `json:"change_type"` // "attribute", "text", "children"
}

// --- Network Capture types ---

// NetworkCapture holds a captured network request/response pair.
type NetworkCapture struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Status          int               `json:"status"`
	MimeType        string            `json:"mime_type,omitempty"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody    string            `json:"response_body,omitempty"`
	Truncated       bool              `json:"truncated,omitempty"`
}

// --- Semantic Form types ---

// FormFieldInfo describes a discovered form field with its label.
type FormFieldInfo struct {
	Selector    string   `json:"selector"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Name        string   `json:"name,omitempty"`
	ID          string   `json:"id,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// FormDiscoveryResult is the structured response from form field discovery.
type FormDiscoveryResult struct {
	FormSelector string          `json:"form_selector"`
	Action       string          `json:"action,omitempty"`
	Method       string          `json:"method,omitempty"`
	Fields       []FormFieldInfo `json:"fields"`
}

// SemanticFillResult is the structured response from semantic form filling.
type SemanticFillResult struct {
	Fields  []SemanticFieldResult `json:"fields"`
	Success bool                  `json:"success"`
}

// --- Visual Grounding types ---

// AnnotatedResult holds an annotated screenshot with element-label mapping.
type AnnotatedResult struct {
	Image    []byte             `json:"-"` // PNG/JPEG image data
	Elements []AnnotatedElement `json:"elements"`
	Count    int                `json:"count"`
}

// AnnotatedElement maps a numbered label to an interactive element.
type AnnotatedElement struct {
	Label    int    `json:"label"`
	Selector string `json:"selector"`
	Tag      string `json:"tag"`
	Type     string `json:"type,omitempty"`
	Text     string `json:"text,omitempty"`
	Href     string `json:"href,omitempty"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// SemanticFieldResult describes the outcome of filling one semantically-matched field.
type SemanticFieldResult struct {
	HumanName string `json:"human_name"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}
