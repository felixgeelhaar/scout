package agent

import "fmt"

// Observe returns a structured snapshot of the page's current state,
// including all interactive elements (links, inputs, buttons).
// This is designed to fit into an agent's context window as a concise
// representation of what actions are available on the page.
func (s *Session) Observe() (*Observation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return nil, err
	}
	return s.observeInternal()
}

// observeInternal is the non-locking implementation of Observe.
// Caller must hold s.mu.
func (s *Session) observeInternal() (*Observation, error) {

	js := `(function() {
		const obs = {
			url: window.location.href,
			title: document.title,
			text: document.body ? document.body.innerText.slice(0, 2000) : '',
			links: [],
			inputs: [],
			buttons: [],
			meta: {}
		};

		// Collect links
		for (const a of document.querySelectorAll('a[href]')) {
			const text = a.textContent.trim();
			if (text || a.href) {
				obs.links.push({text: text.slice(0, 100), href: a.getAttribute('href')});
			}
		}

		// Collect inputs
		for (const input of document.querySelectorAll('input, textarea, select')) {
			obs.inputs.push({
				id: input.id || '',
				name: input.name || '',
				type: input.type || input.tagName.toLowerCase(),
				value: input.value || '',
				placeholder: input.placeholder || ''
			});
		}

		// Collect buttons
		for (const btn of document.querySelectorAll('button, input[type=submit], input[type=button], [role=button]')) {
			obs.buttons.push({
				text: (btn.textContent || btn.value || '').trim().slice(0, 100),
				id: btn.id || '',
				type: btn.type || ''
			});
		}

		// Collect meta tags
		for (const meta of document.querySelectorAll('meta[name], meta[property]')) {
			const key = meta.getAttribute('name') || meta.getAttribute('property');
			const val = meta.getAttribute('content');
			if (key && val) obs.meta[key] = val.slice(0, 200);
		}

		obs.interactive = obs.links.length + obs.inputs.length + obs.buttons.length;
		return obs;
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, fmt.Errorf("agent: failed to observe page: %w", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("agent: unexpected observation result type")
	}

	maxLinks := s.contentOpts.MaxLinks
	if maxLinks == 0 {
		maxLinks = 25
	}
	maxInputs := s.contentOpts.MaxInputs
	if maxInputs == 0 {
		maxInputs = 20
	}
	maxButtons := s.contentOpts.MaxButtons
	if maxButtons == 0 {
		maxButtons = 15
	}

	obs := &Observation{}
	obs.URL, _ = m["url"].(string)
	obs.Title, _ = m["title"].(string)
	obs.Text, _ = m["text"].(string)
	if v, ok := m["interactive"].(float64); ok {
		obs.Interactive = int(v)
	}

	if links, ok := m["links"].([]any); ok {
		for i, l := range links {
			if i >= maxLinks {
				break
			}
			if lm, ok := l.(map[string]any); ok {
				text, _ := lm["text"].(string)
				href, _ := lm["href"].(string)
				obs.Links = append(obs.Links, LinkInfo{Text: text, Href: href})
			}
		}
	}

	if inputs, ok := m["inputs"].([]any); ok {
		for i, inp := range inputs {
			if i >= maxInputs {
				break
			}
			if im, ok := inp.(map[string]any); ok {
				obs.Inputs = append(obs.Inputs, InputInfo{
					ID:          strVal(im, "id"),
					Name:        strVal(im, "name"),
					Type:        strVal(im, "type"),
					Value:       strVal(im, "value"),
					Placeholder: strVal(im, "placeholder"),
				})
			}
		}
	}

	if buttons, ok := m["buttons"].([]any); ok {
		for i, btn := range buttons {
			if i >= maxButtons {
				break
			}
			if bm, ok := btn.(map[string]any); ok {
				obs.Buttons = append(obs.Buttons, ButtonInfo{
					Text: strVal(bm, "text"),
					ID:   strVal(bm, "id"),
					Type: strVal(bm, "type"),
				})
			}
		}
	}

	if meta, ok := m["meta"].(map[string]any); ok {
		obs.Meta = make(map[string]string)
		for k, v := range meta {
			if s, ok := v.(string); ok {
				obs.Meta[k] = s
			}
		}
	}

	return obs, nil
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
