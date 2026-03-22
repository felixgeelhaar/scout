package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/felixgeelhaar/mcp-go"

	browse "github.com/felixgeelhaar/scout"
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
	Fields map[string]string `json:"fields" jsonschema:"required,description=JSON object where keys are CSS selectors and values are text to type. Example: {\"#email\": \"user@test.com\", \"#password\": \"secret\"}"`
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

type ScreenshotInput struct {
	URL string `json:"url,omitempty" jsonschema:"description=Optional URL to navigate to before taking screenshot"`
}

type EvalInput struct {
	Expression string `json:"expression" jsonschema:"required,description=JavaScript expression to evaluate"`
}

type HasElementInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to check for"`
}

type WaitForInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector to wait for"`
}

type ObserveInput struct {
	URL string `json:"url,omitempty" jsonschema:"description=Optional URL to navigate to before observing"`
}

type ObserveWithBudgetInput struct {
	Budget int `json:"budget" jsonschema:"required,description=Approximate token budget for the response"`
}

type PDFInput struct{}

type DiscoverFormInput struct {
	Selector string `json:"selector,omitempty" jsonschema:"description=CSS selector for specific form (empty = all forms)"`
}

type FillFormSemanticInput struct {
	Fields map[string]string `json:"fields" jsonschema:"required,description=JSON object where keys are human-readable field names and values are text to type. Example: {\"Email\": \"user@test.com\", \"Password\": \"secret\"}"`
}

type EnableNetworkInput struct {
	Patterns []string `json:"patterns,omitempty" jsonschema:"description=URL substring patterns to capture (empty = all)"`
}

type NetworkRequestsInput struct {
	Pattern string `json:"pattern,omitempty" jsonschema:"description=URL substring filter"`
}

type AnnotatedScreenshotInput struct {
	IncludeImage bool `json:"include_image,omitempty" jsonschema:"description=Include base64 image data in response. Default false to avoid large responses. Use screenshot tool separately if you need the image."`
}

type AnnotatedScreenshotResult struct {
	Image    string                   `json:"image,omitempty"`
	Elements []agent.AnnotatedElement `json:"elements"`
	Count    int                      `json:"count"`
}

type ClickLabelInput struct {
	Label json.Number `json:"label" jsonschema:"required,description=Label number from annotated screenshot (e.g. 8)"`
}

type ComponentStateInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of the component root element"`
}

type DispatchEventInput struct {
	Selector  string         `json:"selector" jsonschema:"required,description=CSS selector of the target element"`
	EventType string         `json:"event_type" jsonschema:"required,description=DOM event type (e.g. click, input, custom-event)"`
	Detail    map[string]any `json:"detail,omitempty" jsonschema:"description=Event detail/payload data"`
}

type ConfigureInput struct {
	Headless bool `json:"headless" jsonschema:"description=Run browser in headless mode (no visible window). Default true."`
}

type HoverInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of element to hover over"`
}

type SelectOptionInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of the select element"`
	Option   string `json:"option" jsonschema:"required,description=Option text or value to select"`
}

type ScrollToInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of element to scroll into view"`
}

type ScrollByInput struct {
	X int `json:"x" jsonschema:"description=Horizontal scroll offset in pixels"`
	Y int `json:"y" jsonschema:"required,description=Vertical scroll offset in pixels (positive=down)"`
}

type DragDropInput struct {
	From string `json:"from" jsonschema:"required,description=CSS selector of element to drag"`
	To   string `json:"to" jsonschema:"required,description=CSS selector of drop target"`
}

type FocusInput struct {
	Selector string `json:"selector" jsonschema:"required,description=CSS selector of element to focus"`
}

type StartRecordInput struct {
	Name string `json:"name" jsonschema:"required,description=Name for the playbook being recorded"`
}

type SavePlaybookInput struct {
	Path string `json:"path" jsonschema:"required,description=File path to save the playbook JSON"`
}

type ReplayInput struct {
	Path string `json:"path" jsonschema:"required,description=File path to the playbook JSON file"`
}

type TabInput struct {
	Name string `json:"name,omitempty" jsonschema:"description=Tab name (for open_tab and switch_tab)"`
	URL  string `json:"url,omitempty" jsonschema:"description=URL to open in new tab"`
}

type HistoryInput struct {
	Count int `json:"count,omitempty" jsonschema:"description=Number of recent actions to return (default 5, max 20)"`
}

type SuggestInput struct {
	Selector string `json:"selector" jsonschema:"required,description=The selector that failed to match"`
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

	// Lazy session — created on first tool use, can be reconfigured without restart.
	// All handlers reference `session` which is lazily initialized via ensureSession().
	var (
		session    *agent.Session
		sessionCfg = agent.SessionConfig{Headless: true}
		sessionMu  sync.Mutex
	)

	ensureSession := func() error {
		sessionMu.Lock()
		defer sessionMu.Unlock()
		if session != nil {
			return nil
		}
		s, err := agent.NewSession(sessionCfg)
		if err != nil {
			return err
		}
		session = s
		return nil
	}

	reconfigure := func(cfg agent.SessionConfig) error {
		sessionMu.Lock()
		defer sessionMu.Unlock()
		if session != nil {
			// Close in goroutine with timeout — don't block if CDP calls are in flight
			old := session
			session = nil
			go func() {
				done := make(chan struct{})
				go func() { _ = old.Close(); close(done) }()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					// Force abandon — the browser process will be cleaned up by OS
				}
			}()
		}
		sessionCfg = cfg
		return nil
	}

	defer func() {
		sessionMu.Lock()
		if session != nil {
			_ = session.Close()
		}
		sessionMu.Unlock()
	}()

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
see only what changed. Use 'annotated_screenshot' for visual element identification.
Use 'configure' to switch between headless and visible browser modes without restarting.

IMPORTANT: Scout uses standard CSS selectors, NOT Playwright selectors. Do NOT use :text(), :has-text(), >> chaining, or other Playwright-specific syntax. Instead:
- To find by text content: use 'observe' or 'annotated_screenshot' to discover elements, then click by selector or label number
- To find a button by text: use 'fill_form_semantic' for forms, or call 'annotated_screenshot' and use 'click_label' with the label number
- Valid selectors: #id, .class, tag, [attr=value], tag:nth-of-type(n), tag:first-child, etc.

IMPORTANT: fill_form and fill_form_semantic take a JSON OBJECT (not array) for fields:
  fill_form: {"fields": {"#email": "value", "#password": "value"}}
  fill_form_semantic: {"fields": {"Email": "value", "Password": "value"}}
Do NOT send fields as an array of objects.

WORKFLOW: navigate first, then use other tools. Use 'dismiss_cookies' after navigate if a cookie banner appears. Use 'check_readiness' if the page seems to still be loading.`))

	// s returns the current session, lazily creating it on first use.
	// Every handler calls this instead of accessing session directly.
	s := func() *agent.Session {
		if err := ensureSession(); err != nil {
			panic(fmt.Sprintf("failed to create browser session: %v", err))
		}
		return session
	}

	// maybeNavigate navigates if a URL is provided, otherwise uses the current page.
	maybeNavigate := func(url string) error {
		if url != "" {
			_, err := s().Navigate(url)
			return err
		}
		return nil
	}

	// --- Configuration ---

	srv.Tool("configure").
		ClosedWorld().Idempotent().
		Description("Change browser settings without restarting. Use headless=false to see the browser window.").
		Handler(func(ctx context.Context, input ConfigureInput) (string, error) {
			if err := reconfigure(agent.SessionConfig{
				Headless: input.Headless,
			}); err != nil {
				return "", err
			}
			mode := "headless"
			if !input.Headless {
				mode = "visible"
			}
			return fmt.Sprintf("Browser reconfigured: %s mode. Next navigation will use the new settings.", mode), nil
		})

	// --- Navigation & Observation ---

	srv.Tool("navigate").
		OpenWorld().
		Description("Navigate to a URL. Returns page title and URL.").
		Handler(func(ctx context.Context, input NavigateInput) (*agent.PageResult, error) {
			progress := mcp.ProgressFromContext(ctx)
			total := 3.0
			_ = progress.ReportWithMessage(1, &total, "Launching browser...")
			result, err := s().Navigate(input.URL)
			if err != nil {
				return nil, err
			}
			_ = progress.ReportWithMessage(2, &total, "Page loaded")
			_ = progress.ReportWithMessage(3, &total, "Done")
			// Notify client that available tools may have changed based on page content
			if sess := mcp.SessionFromContext(ctx); sess != nil {
				_ = sess.NotifyToolListChanged()
			}
			// Push page info via channel if supported
			if ch := mcp.ChannelFromContext(ctx); ch != nil {
				_ = ch.SendText("scout.navigation", fmt.Sprintf("Navigated to %s — %s", result.URL, result.Title))
			}
			return result, nil
		})

	srv.Tool("observe").
		ReadOnly().
		OutputSchema(agent.Observation{}).
		Description("Get a structured snapshot of the current page. Optionally pass url to navigate first.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.Observation, error) {
			if err := maybeNavigate(input.URL); err != nil {
				return nil, err
			}
			return s().Observe()
		})

	srv.Tool("observe_diff").
		ReadOnly().
		Description("Observe the page and return only what changed since the last observation. Much more token-efficient.").
		Handler(func(ctx context.Context, input ObserveInput) (*ObserveDiffResult, error) {
			obs, diff, err := s().ObserveDiff()
			if err != nil {
				return nil, err
			}
			return &ObserveDiffResult{Observation: obs, Diff: diff}, nil
		})

	srv.Tool("observe_with_budget").
		ReadOnly().
		Description("Observe the page constrained to an approximate token budget. Prioritizes interactive elements.").
		Handler(func(ctx context.Context, input ObserveWithBudgetInput) (*agent.Observation, error) {
			return s().ObserveWithBudget(input.Budget)
		})

	// --- Interaction ---

	srv.Tool("click").
		Description("Click an element by CSS selector. Set wait=true for navigation clicks.").
		Handler(func(ctx context.Context, input ClickInput) (*agent.PageResult, error) {
			if input.Wait {
				return s().ClickAndWait(input.Selector)
			}
			return s().Click(input.Selector)
		})

	srv.Tool("click_label").
		Description("Click an element by its label number from annotated_screenshot.").
		Handler(func(ctx context.Context, input ClickLabelInput) (*agent.PageResult, error) {
			label, err := strconv.Atoi(input.Label.String())
			if err != nil {
				return nil, fmt.Errorf("label must be a number (e.g. 8), got %q", input.Label.String())
			}
			return s().ClickLabel(label)
		})

	srv.Tool("type").
		Description("Type text into an input element. Clears existing value first.").
		Handler(func(ctx context.Context, input TypeInput) (*agent.ElementResult, error) {
			return s().Type(input.Selector, input.Text)
		})

	srv.Tool("fill_form").
		Description("Fill multiple form fields at once. Keys are CSS selectors, values are text to type.").
		Handler(func(ctx context.Context, input FillFormInput) (*agent.FormResult, error) {
			return s().FillForm(input.Fields)
		})

	srv.Tool("fill_form_semantic").
		Description("Fill form fields using human-readable names (e.g., 'Email', 'Password') instead of CSS selectors.").
		Handler(func(ctx context.Context, input FillFormSemanticInput) (*agent.SemanticFillResult, error) {
			return s().FillFormSemantic(input.Fields)
		})

	srv.Tool("dispatch_event").
		Description("Dispatch a DOM event on an element. Useful for triggering SPA event handlers.").
		Handler(func(ctx context.Context, input DispatchEventInput) (string, error) {
			if err := s().DispatchEvent(input.Selector, input.EventType, input.Detail); err != nil {
				return "", err
			}
			return fmt.Sprintf("Dispatched %s on %s", input.EventType, input.Selector), nil
		})

	srv.Tool("hover").
		Description("Hover over an element to trigger CSS :hover states, tooltips, and dropdown menus.").
		Handler(func(ctx context.Context, input HoverInput) (*agent.PageResult, error) {
			return s().Hover(input.Selector)
		})

	srv.Tool("double_click").
		Description("Double-click an element.").
		Handler(func(ctx context.Context, input ClickInput) (*agent.PageResult, error) {
			return s().DoubleClick(input.Selector)
		})

	srv.Tool("right_click").
		Description("Right-click an element to trigger context menus.").
		Handler(func(ctx context.Context, input ClickInput) (*agent.PageResult, error) {
			return s().RightClick(input.Selector)
		})

	srv.Tool("select_option").
		Description("Select an option from a dropdown/select element by visible text or value.").
		Handler(func(ctx context.Context, input SelectOptionInput) (*agent.ElementResult, error) {
			return s().SelectOption(input.Selector, input.Option)
		})

	srv.Tool("scroll_to").
		Description("Scroll to bring an element into view.").
		Handler(func(ctx context.Context, input ScrollToInput) (*agent.PageResult, error) {
			return s().ScrollTo(input.Selector)
		})

	srv.Tool("scroll_by").
		Description("Scroll the page by pixel offset. Positive y = scroll down.").
		Handler(func(ctx context.Context, input ScrollByInput) (*agent.PageResult, error) {
			return s().ScrollBy(input.X, input.Y)
		})

	srv.Tool("focus").
		Description("Set focus on an element, triggering :focus CSS state.").
		Handler(func(ctx context.Context, input FocusInput) (*agent.PageResult, error) {
			return s().Focus(input.Selector)
		})

	srv.Tool("drag_drop").
		Description("Drag an element and drop it on another element.").
		Handler(func(ctx context.Context, input DragDropInput) (*agent.PageResult, error) {
			return s().DragDrop(input.From, input.To)
		})

	// --- Extraction ---

	srv.Tool("extract").
		ReadOnly().
		Description("Extract text content from a single element.").
		Handler(func(ctx context.Context, input ExtractInput) (*agent.ElementResult, error) {
			return s().Extract(input.Selector)
		})

	srv.Tool("extract_all").
		ReadOnly().
		Description("Extract text from all elements matching a selector.").
		Handler(func(ctx context.Context, input ExtractAllInput) (*agent.ExtractAllResult, error) {
			return s().ExtractAll(input.Selector)
		})

	srv.Tool("extract_table").
		ReadOnly().
		Description("Extract structured data from an HTML table (headers + rows).").
		Handler(func(ctx context.Context, input ExtractTableInput) (*agent.TableResult, error) {
			return s().ExtractTable(input.Selector)
		})

	srv.Tool("markdown").
		ReadOnly().
		Description("Get a compact markdown representation of the page. Ideal for LLM processing.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return s().Markdown()
		})

	srv.Tool("readable_text").
		ReadOnly().
		Description("Extract just the main readable content, stripping navigation and boilerplate.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return s().ReadableText()
		})

	srv.Tool("accessibility_tree").
		ReadOnly().
		Description("Get a compact accessibility tree showing all interactive elements.").
		Handler(func(ctx context.Context, input ObserveInput) (string, error) {
			return s().AccessibilityTree()
		})

	// --- Capture ---

	srv.Tool("screenshot").
		ReadOnly().
		Description("Capture a PNG screenshot. Optionally pass url to navigate first. Auto-compressed for LLM contexts.").
		Handler(func(ctx context.Context, input ScreenshotInput) (string, error) {
			if err := maybeNavigate(input.URL); err != nil {
				return "", err
			}
			// Use aggressive compression for MCP — 200KB max to avoid blowing LLM context
			page := s().Page()
			if page == nil {
				return "", fmt.Errorf("no page open")
			}
			data, err := page.ScreenshotWithOptions(browse.ScreenshotOptions{
				MaxSize: 200 * 1024, // 200KB — ~2.5k tokens in base64
			})
			if err != nil {
				return "", err
			}
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
		})

	srv.Tool("annotated_screenshot").
		ReadOnly().
		Description("Label all interactive elements with numbers and return their selectors/info. By default returns only the element list (compact). Set include_image=true to also get the screenshot with labels drawn on it.").
		Handler(func(ctx context.Context, input AnnotatedScreenshotInput) (*AnnotatedScreenshotResult, error) {
			result, err := s().AnnotatedScreenshot()
			if err != nil {
				return nil, err
			}
			out := &AnnotatedScreenshotResult{
				Elements: result.Elements,
				Count:    result.Count,
			}
			if input.IncludeImage {
				out.Image = "data:image/png;base64," + base64.StdEncoding.EncodeToString(result.Image)
			}
			return out, nil
		})

	srv.Tool("pdf").
		ReadOnly().
		Description("Generate a PDF of the current page.").
		Handler(func(ctx context.Context, input PDFInput) (string, error) {
			data, err := s().PDF()
			if err != nil {
				return "", err
			}
			return "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(data), nil
		})

	// --- Network ---

	srv.Tool("enable_network_capture").
		ClosedWorld().
		Description("Start capturing network XHR/fetch responses. Patterns filter by URL substring.").
		Handler(func(ctx context.Context, input EnableNetworkInput) (string, error) {
			if err := s().EnableNetworkCapture(input.Patterns...); err != nil {
				return "", err
			}
			return "Network capture enabled", nil
		})

	srv.Tool("network_requests").
		ReadOnly().
		Description("Get captured network requests/responses. Call enable_network_capture first.").
		Handler(func(ctx context.Context, input NetworkRequestsInput) ([]agent.NetworkCapture, error) {
			return s().CapturedRequests(input.Pattern), nil
		})

	// --- Framework support ---

	srv.Tool("wait_spa").
		ReadOnly().
		Description("Wait for SPA framework (React/Vue/Angular/Next.js/Svelte) to finish rendering.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.PageResult, error) {
			if err := s().WaitForSPA(); err != nil {
				return nil, err
			}
			return s().Snapshot()
		})

	srv.Tool("detect_frameworks").
		ReadOnly().
		Description("Detect which frontend frameworks are active on the page.").
		Handler(func(ctx context.Context, input ObserveInput) ([]string, error) {
			return s().DetectedFrameworks()
		})

	srv.Tool("component_state").
		ReadOnly().
		Description("Extract component state/props from any framework (React, Vue, Svelte, Angular, Alpine, Lit).").
		Handler(func(ctx context.Context, input ComponentStateInput) (map[string]any, error) {
			return s().ComponentState(input.Selector)
		})

	srv.Tool("app_state").
		ReadOnly().
		Description("Extract global app state (Redux, Next.js, Nuxt, Remix, SvelteKit, Gatsby, Astro, Alpine, HTMX).").
		Handler(func(ctx context.Context, input ObserveInput) (map[string]any, error) {
			return s().GetAppState()
		})

	// --- Dialog detection ---

	srv.Tool("detect_dialog").
		ReadOnly().
		Description("Check if a modal, dialog, popup, or overlay is currently visible. Returns its title, text, buttons, and inputs.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.DialogInfo, error) {
			return s().DetectDialog()
		})

	// --- Smart helpers ---

	srv.Tool("dismiss_cookies").
		Description("Auto-dismiss cookie consent banners. Tries common selectors and text patterns (Accept, Agree, Got it, OK). Returns whether a banner was found and dismissed.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.CookieDismissResult, error) {
			return s().DismissCookieBanner()
		})

	srv.Tool("check_readiness").
		ReadOnly().
		Description("Check how ready the page is for interaction. Returns a 0-100 score, pending XHR count, skeleton/spinner presence, and suggestions for what to wait for.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.PageReadiness, error) {
			return s().CheckReadiness()
		})

	srv.Tool("suggest_selectors").
		ReadOnly().
		Description("Find elements similar to a selector that failed. Returns up to 5 suggestions with selector, tag, text, and classes.").
		Handler(func(ctx context.Context, input SuggestInput) ([]agent.SelectorSuggestion, error) {
			return s().SuggestSelectors(input.Selector)
		})

	srv.Tool("session_history").
		ReadOnly().
		Description("Get the last N actions performed in this session. Provides context about what has been done so far.").
		Handler(func(ctx context.Context, input HistoryInput) ([]agent.HistoryEntry, error) {
			count := input.Count
			if count == 0 {
				count = 5
			}
			return s().SessionHistory(count), nil
		})

	// --- Smart extraction ---

	srv.Tool("auto_extract").
		ReadOnly().
		Description("Auto-detect repeating patterns (product cards, search results, list items) and extract structured data. No selectors needed.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.ExtractedPattern, error) {
			return s().AutoExtract()
		})

	srv.Tool("scroll_and_collect").
		Description("Auto-scroll the page and collect items as they lazy-load. For infinite scroll pages.").
		Handler(func(ctx context.Context, input struct {
			Selector string `json:"selector" jsonschema:"required,description=CSS selector for the repeating items"`
			MaxItems int    `json:"max_items,omitempty" jsonschema:"description=Maximum items to collect (default 100)"`
		}) (*agent.ExtractAllResult, error) {
			return s().ScrollAndCollect(input.Selector, input.MaxItems)
		})

	// --- Diagnostics ---

	srv.Tool("console_errors").
		ReadOnly().
		Description("Get captured console.error and console.warn messages from the page. Helps debug broken pages.").
		Handler(func(ctx context.Context, input ObserveInput) ([]agent.ConsoleMessage, error) {
			return s().ConsoleErrors()
		})

	srv.Tool("detect_auth_wall").
		ReadOnly().
		Description("Check if the page is a login wall, paywall, or CAPTCHA. Returns type, confidence, and reason.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.AuthWallResult, error) {
			return s().DetectAuthWall()
		})

	srv.Tool("upload_file").
		Description("Upload a file to a file input element.").
		Handler(func(ctx context.Context, input struct {
			Selector string `json:"selector" jsonschema:"required,description=CSS selector of the file input element"`
			FilePath string `json:"file_path" jsonschema:"required,description=Local path to the file to upload"`
		}) (string, error) {
			if err := s().UploadFile(input.Selector, input.FilePath); err != nil {
				return "", err
			}
			return fmt.Sprintf("Uploaded %s to %s", input.FilePath, input.Selector), nil
		})

	srv.Tool("compare_tabs").
		ReadOnly().
		Description("Compare content between two named tabs. Returns what's different, what's only in one tab.").
		Handler(func(ctx context.Context, input struct {
			Tab1 string `json:"tab1" jsonschema:"required,description=Name of the first tab"`
			Tab2 string `json:"tab2" jsonschema:"required,description=Name of the second tab"`
		}) (*agent.PageDiff, error) {
			return s().CompareTabs(input.Tab1, input.Tab2)
		})

	// --- Utility ---

	srv.Tool("has_element").
		ReadOnly().
		Description("Check if an element exists on the page.").
		Handler(func(ctx context.Context, input HasElementInput) (bool, error) {
			return s().HasElement(input.Selector), nil
		})

	srv.Tool("wait_for").
		ReadOnly().
		Description("Wait for an element to appear in the DOM.").
		Handler(func(ctx context.Context, input WaitForInput) (*agent.PageResult, error) {
			if err := s().WaitFor(input.Selector); err != nil {
				return nil, err
			}
			return s().Snapshot()
		})

	srv.Tool("discover_form").
		ReadOnly().
		Description("Discover form fields with their labels, types, and CSS selectors.").
		Handler(func(ctx context.Context, input DiscoverFormInput) (*agent.FormDiscoveryResult, error) {
			return s().DiscoverForm(input.Selector)
		})

	// --- Tabs ---

	srv.Tool("open_tab").
		OpenWorld().
		Description("Open a new named browser tab and navigate to a URL. The new tab becomes active.").
		Handler(func(ctx context.Context, input TabInput) (*agent.PageResult, error) {
			return s().OpenTab(input.Name, input.URL)
		})

	srv.Tool("switch_tab").
		ClosedWorld().
		Description("Switch to a named tab. Use list_tabs to see available tabs.").
		Handler(func(ctx context.Context, input TabInput) (*agent.PageResult, error) {
			return s().SwitchTab(input.Name)
		})

	srv.Tool("close_tab").
		ClosedWorld().
		Description("Close a named tab. Cannot close the currently active tab.").
		Handler(func(ctx context.Context, input TabInput) (string, error) {
			if err := s().CloseTab(input.Name); err != nil {
				return "", err
			}
			return fmt.Sprintf("Closed tab %q", input.Name), nil
		})

	srv.Tool("list_tabs").
		ReadOnly().
		Description("List all open tabs with their names, URLs, and titles.").
		Handler(func(ctx context.Context, input ObserveInput) ([]agent.TabInfo, error) {
			return s().ListTabs()
		})

	// --- Playbook ---

	srv.Tool("start_recording").
		ClosedWorld().
		Description("Start recording browser actions into a replayable playbook. Call stop_recording when done.").
		Handler(func(ctx context.Context, input StartRecordInput) (string, error) {
			s().StartRecordingPlaybook(input.Name)
			return fmt.Sprintf("Recording started: %s", input.Name), nil
		})

	srv.Tool("stop_recording").
		ClosedWorld().
		Description("Stop recording and return the playbook. Save it with save_playbook for later replay.").
		Handler(func(ctx context.Context, input ObserveInput) (*agent.Playbook, error) {
			return s().StopRecordingPlaybook()
		})

	srv.Tool("save_playbook").
		ClosedWorld().
		Description("Save the last recorded playbook to a JSON file for deterministic replay.").
		Handler(func(ctx context.Context, input SavePlaybookInput) (string, error) {
			pb, err := s().StopRecordingPlaybook()
			if err != nil {
				return "", err
			}
			if err := agent.SavePlaybook(pb, input.Path); err != nil {
				return "", err
			}
			return fmt.Sprintf("Playbook saved to %s (%d actions)", input.Path, len(pb.Actions)), nil
		})

	srv.Tool("replay_playbook").
		OpenWorld().
		Description("Replay a saved playbook deterministically without LLM calls. Returns success/failure and any extracted data.").
		Handler(func(ctx context.Context, input ReplayInput) (*agent.PlaybookResult, error) {
			pb, err := agent.LoadPlaybook(input.Path)
			if err != nil {
				return nil, err
			}
			return s().ReplayPlaybook(pb)
		})

	// --- Gated tools ---

	if os.Getenv("SCOUT_ENABLE_EVAL") == "1" {
		srv.Tool("eval").
			Description("Execute JavaScript on the page. WARNING: arbitrary code execution.").
			Handler(func(ctx context.Context, input EvalInput) (any, error) {
				return s().Eval(input.Expression)
			})
	}

	if err := mcp.ServeStdio(ctx, srv); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
