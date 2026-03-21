package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/mcp-go"

	"github.com/felixgeelhaar/scout/agent"
)

// --- Tool input types ---

type NavigateInput struct {
	URL string `json:"url" jsonschema:"required,description=URL to navigate to"`
}

type ClickInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of element to click"`
	Wait     bool   `json:"wait,omitempty" jsonschema:"description=If true wait for full page navigation after click"`
}

type TypeInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of input element"`
	Text     string `json:"text" jsonschema:"required,description=Text to type into the element"`
}

type FillFormInput struct {
	Fields map[string]string `json:"fields" jsonschema:"required,description=Map of CSS selector to value for each field"`
}

type ExtractInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to extract text from"`
}

type ExtractAllInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to extract all matching texts"`
}

type ExtractTableInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector for the table element"`
}

type ScreenshotInput struct{}

type EvalInput struct {
	Expression string `json:"expression" jsonschema:"required,description=JavaScript expression to evaluate"`
}

type HasElementInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to check for"`
}

type WaitForInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to wait for"`
}

type ObserveInput struct{}

type ObserveWithBudgetInput struct {
	Budget int `json:"budget" jsonschema:"required,description=Approximate token budget for the response"`
}

type PDFInput struct{}

type DiscoverFormInput struct {
	Selector string `json:"selector,omitempty" jsonschema:"description=CSS selector for specific form (empty = all forms)"`
}

type FillFormSemanticInput struct {
	Fields map[string]string `json:"fields" jsonschema:"required,description=Map of human-readable field name to value"`
}

type EnableNetworkInput struct {
	Patterns []string `json:"patterns,omitempty" jsonschema:"description=URL substring patterns to capture (empty = all)"`
}

type NetworkRequestsInput struct {
	Pattern string `json:"pattern,omitempty" jsonschema:"description=URL substring filter"`
}

type AnnotatedScreenshotResult struct {
	Image    string                   `json:"image"`
	Elements []agent.AnnotatedElement `json:"elements"`
	Count    int                      `json:"count"`
}

type ClickLabelInput struct {
	Label int `json:"label" jsonschema:"required,description=Label number from annotated screenshot"`
}

type ComponentStateInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of the component root element"`
}

type DispatchEventInput struct {
	Selector  string         `json:"selector" jsonschema:"required,description=CSS selector of the target element"`
	EventType string         `json:"event_type" jsonschema:"required,description=DOM event type (e.g. click, input, custom-event)"`
	Detail    map[string]any `json:"detail,omitempty" jsonschema:"description=Event detail/payload data"`
}

type ObserveDiffResult struct {
	Observation *agent.Observation `json:"observation"`
	Diff        *agent.DOMDiff     `json:"diff"`
}

func serveMCP() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	session, err := agent.NewSession(agent.SessionConfig{
		Headless: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = session.Close() }()

	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "scout",
		Version: version,
		Capabilities: mcp.Capabilities{
			Tools: true,
		},
	}, mcp.WithInstructions(`Scout provides browser automation tools for navigating websites,
filling forms, extracting data, and taking screenshots. Start with 'navigate' to load a page,
then use 'observe' to see interactive elements, and perform actions with 'click', 'type',
'fill_form_semantic', 'extract', 'extract_table', etc. Use 'observe_diff' after actions to
see only what changed. Use 'annotated_screenshot' for visual element identification.`))

	// --- Navigation & Observation ---

	srv.Tool("navigate").
		Description("Navigate to a URL. Returns page title and URL.").
		Handler(func(ctx context.Context, input NavigateInput) (*agent.PageResult, error) {
			return session.Navigate(input.URL)
		})

	srv.Tool("observe").
		Description("Get a structured snapshot of the current page including all links, inputs, buttons, and visible text.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.Observation, error) {
			return session.Observe()
		})

	srv.Tool("observe_diff").
		Description("Observe the page and return only what changed since the last observation. Much more token-efficient.").
		Handler(func(ctx context.Context, input ObserveInput) (*ObserveDiffResult, error) {
			obs, diff, err := session.ObserveDiff()
			if err != nil {
				return nil, err
			}
			return &ObserveDiffResult{Observation: obs, Diff: diff}, nil
		})

	srv.Tool("observe_with_budget").
		Description("Observe the page constrained to an approximate token budget. Prioritizes interactive elements.").
		Handler(func(ctx context.Context, input ObserveWithBudgetInput) (*agent.Observation, error) {
			return session.ObserveWithBudget(input.Budget)
		})

	// --- Interaction ---

	srv.Tool("click").
		Description("Click an element by CSS selector. Set wait=true for navigation clicks.").
		Handler(func(ctx context.Context, input ClickInput) (*agent.PageResult, error) {
			if input.Wait {
				return session.ClickAndWait(input.Selector)
			}
			return session.Click(input.Selector)
		})

	srv.Tool("click_label").
		Description("Click an element by its label number from annotated_screenshot.").
		Handler(func(ctx context.Context, input ClickLabelInput) (*agent.PageResult, error) {
			return session.ClickLabel(input.Label)
		})

	srv.Tool("type").
		Description("Type text into an input element. Clears existing value first.").
		Handler(func(ctx context.Context, input TypeInput) (*agent.ElementResult, error) {
			return session.Type(input.Selector, input.Text)
		})

	srv.Tool("fill_form").
		Description("Fill multiple form fields at once. Keys are CSS selectors, values are text to type.").
		Handler(func(ctx context.Context, input FillFormInput) (*agent.FormResult, error) {
			return session.FillForm(input.Fields)
		})

	srv.Tool("fill_form_semantic").
		Description("Fill form fields using human-readable names (e.g., 'Email', 'Password') instead of CSS selectors.").
		Handler(func(ctx context.Context, input FillFormSemanticInput) (*agent.SemanticFillResult, error) {
			return session.FillFormSemantic(input.Fields)
		})

	srv.Tool("dispatch_event").
		Description("Dispatch a DOM event on an element. Useful for triggering SPA event handlers.").
		Handler(func(ctx context.Context, input DispatchEventInput) (string, error) {
			if err := session.DispatchEvent(input.Selector, input.EventType, input.Detail); err != nil {
				return "", err
			}
			return fmt.Sprintf("Dispatched %s on %s", input.EventType, input.Selector), nil
		})

	// --- Extraction ---

	srv.Tool("extract").
		Description("Extract text content from a single element.").
		Handler(func(ctx context.Context, input ExtractInput) (*agent.ElementResult, error) {
			return session.Extract(input.Selector)
		})

	srv.Tool("extract_all").
		Description("Extract text from all elements matching a selector.").
		Handler(func(ctx context.Context, input ExtractAllInput) (*agent.ExtractAllResult, error) {
			return session.ExtractAll(input.Selector)
		})

	srv.Tool("extract_table").
		Description("Extract structured data from an HTML table (headers + rows).").
		Handler(func(ctx context.Context, input ExtractTableInput) (*agent.TableResult, error) {
			return session.ExtractTable(input.Selector)
		})

	srv.Tool("markdown").
		Description("Get a compact markdown representation of the page. Ideal for LLM processing.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return session.Markdown()
		})

	srv.Tool("readable_text").
		Description("Extract just the main readable content, stripping navigation and boilerplate.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return session.ReadableText()
		})

	srv.Tool("accessibility_tree").
		Description("Get a compact accessibility tree showing all interactive elements.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return session.AccessibilityTree()
		})

	// --- Capture ---

	srv.Tool("screenshot").
		Description("Capture a PNG screenshot of the current page (auto-compressed to fit LLM contexts).").
		Handler(func(ctx context.Context, input ScreenshotInput) (string, error) {
			data, err := session.Screenshot()
			if err != nil {
				return "", err
			}
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
		})

	srv.Tool("annotated_screenshot").
		Description("Screenshot with numbered labels on interactive elements. Use click_label to interact by number.").
		Handler(func(ctx context.Context, input ObserveInput) (*AnnotatedScreenshotResult, error) {
			result, err := session.AnnotatedScreenshot()
			if err != nil {
				return nil, err
			}
			return &AnnotatedScreenshotResult{
				Image:    "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Image),
				Elements: result.Elements,
				Count:    result.Count,
			}, nil
		})

	srv.Tool("pdf").
		Description("Generate a PDF of the current page.").
		Handler(func(ctx context.Context, input PDFInput) (string, error) {
			data, err := session.PDF()
			if err != nil {
				return "", err
			}
			return "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(data), nil
		})

	// --- Network ---

	srv.Tool("enable_network_capture").
		Description("Start capturing network XHR/fetch responses. Patterns filter by URL substring.").
		Handler(func(ctx context.Context, input EnableNetworkInput) (string, error) {
			if err := session.EnableNetworkCapture(input.Patterns...); err != nil {
				return "", err
			}
			return "Network capture enabled", nil
		})

	srv.Tool("network_requests").
		Description("Get captured network requests/responses. Call enable_network_capture first.").
		Handler(func(ctx context.Context, input NetworkRequestsInput) ([]agent.NetworkCapture, error) {
			return session.CapturedRequests(input.Pattern), nil
		})

	// --- Framework support ---

	srv.Tool("wait_spa").
		Description("Wait for SPA framework (React/Vue/Angular/Next.js/Svelte) to finish rendering.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.PageResult, error) {
			if err := session.WaitForSPA(); err != nil {
				return nil, err
			}
			return session.Snapshot()
		})

	srv.Tool("detect_frameworks").
		Description("Detect which frontend frameworks are active on the page.").
		Handler(func(ctx context.Context, input ObserveInput) ([]string, error) {
			return session.DetectedFrameworks()
		})

	srv.Tool("component_state").
		Description("Extract component state/props from any framework (React, Vue, Svelte, Angular, Alpine, Lit).").
		Handler(func(ctx context.Context, input ComponentStateInput) (map[string]any, error) {
			return session.ComponentState(input.Selector)
		})

	srv.Tool("app_state").
		Description("Extract global app state (Redux, Next.js, Nuxt, Remix, SvelteKit, Gatsby, Astro, Alpine, HTMX).").
		Handler(func(ctx context.Context, input ObserveInput) (map[string]any, error) {
			return session.GetAppState()
		})

	// --- Utility ---

	srv.Tool("has_element").
		Description("Check if an element exists on the page.").
		Handler(func(ctx context.Context, input HasElementInput) (bool, error) {
			return session.HasElement(input.Selector), nil
		})

	srv.Tool("wait_for").
		Description("Wait for an element to appear in the DOM.").
		Handler(func(ctx context.Context, input WaitForInput) (*agent.PageResult, error) {
			if err := session.WaitFor(input.Selector); err != nil {
				return nil, err
			}
			return session.Snapshot()
		})

	srv.Tool("discover_form").
		Description("Discover form fields with their labels, types, and CSS selectors.").
		Handler(func(ctx context.Context, input DiscoverFormInput) (*agent.FormDiscoveryResult, error) {
			return session.DiscoverForm(input.Selector)
		})

	// --- Gated tools ---

	if os.Getenv("SCOUT_ENABLE_EVAL") == "1" {
		srv.Tool("eval").
			Description("Execute JavaScript on the page. WARNING: arbitrary code execution.").
			Handler(func(ctx context.Context, input EvalInput) (any, error) {
				return session.Eval(input.Expression)
			})
	}

	if err := mcp.ServeStdio(ctx, srv); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
