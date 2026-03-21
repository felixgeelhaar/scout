package agent

import (
	"encoding/json"
	"fmt"
)

// ObserveDiff returns the current page observation along with a structured diff
// of what changed since the last observation. On the first call, the diff is empty.
// This is much more token-efficient than re-sending the full page state each time.
func (s *Session) ObserveDiff() (*Observation, *DOMDiff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, nil, err
	}

	// Install mutation observer if not already active
	if !s.diffInstalled {
		if err := s.installMutationObserver(); err != nil {
			return nil, nil, fmt.Errorf("agent: failed to install mutation observer: %w", err)
		}
		s.diffInstalled = true
	}

	// Harvest mutations
	diff, err := s.harvestMutations()
	if err != nil {
		diff = &DOMDiff{}
	}

	// Get current observation (call internal version without lock)
	obs, err := s.observeInternal()
	if err != nil {
		return nil, nil, err
	}

	return obs, diff, nil
}

func (s *Session) installMutationObserver() error {
	js := `(function() {
		if (window.__browseMutations) return true;
		window.__browseMutations = [];
		const observer = new MutationObserver(mutations => {
			for (const m of mutations) {
				const record = { type: m.type };
				if (m.type === 'childList') {
					record.added = Array.from(m.addedNodes)
						.filter(n => n.nodeType === 1)
						.map(n => ({
							tag: n.tagName.toLowerCase(),
							id: n.id || '',
							classes: n.className || '',
							text: (n.textContent || '').slice(0, 200)
						}));
					record.removed = Array.from(m.removedNodes)
						.filter(n => n.nodeType === 1)
						.map(n => ({
							tag: n.tagName.toLowerCase(),
							id: n.id || '',
							classes: n.className || '',
							text: (n.textContent || '').slice(0, 200)
						}));
				} else if (m.type === 'attributes') {
					record.tag = m.target.tagName ? m.target.tagName.toLowerCase() : '';
					record.id = m.target.id || '';
					record.attribute = m.attributeName;
					record.oldValue = m.oldValue || '';
					record.newValue = m.target.getAttribute(m.attributeName) || '';
				} else if (m.type === 'characterData') {
					record.tag = 'text';
					record.oldValue = (m.oldValue || '').slice(0, 200);
					record.newValue = (m.target.textContent || '').slice(0, 200);
				}
				if (window.__browseMutations.length < 500) {
					window.__browseMutations.push(record);
				}
			}
		});
		observer.observe(document.body || document.documentElement, {
			childList: true, subtree: true, attributes: true,
			attributeOldValue: true, characterData: true, characterDataOldValue: true
		});
		return true;
	})()`

	_, err := s.page.Evaluate(js)
	return err
}

func (s *Session) harvestMutations() (*DOMDiff, error) {
	js := `(function() {
		const mutations = window.__browseMutations || [];
		window.__browseMutations = [];
		return JSON.stringify(mutations);
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}

	str, ok := result.(string)
	if !ok || str == "" || str == "[]" {
		return &DOMDiff{HasDiff: false}, nil
	}

	var rawMutations []map[string]any
	if err := json.Unmarshal([]byte(str), &rawMutations); err != nil {
		return &DOMDiff{HasDiff: false}, nil
	}

	diff := &DOMDiff{}

	for _, m := range rawMutations {
		typ, _ := m["type"].(string)
		switch typ {
		case "childList":
			if added, ok := m["added"].([]any); ok {
				for _, a := range added {
					if el, ok := a.(map[string]any); ok {
						diff.Added = append(diff.Added, parseDOMElement(el))
					}
				}
			}
			if removed, ok := m["removed"].([]any); ok {
				for _, r := range removed {
					if el, ok := r.(map[string]any); ok {
						diff.Removed = append(diff.Removed, parseDOMElement(el))
					}
				}
			}
		case "attributes":
			diff.Modified = append(diff.Modified, DOMChange{
				Tag:        strVal(m, "tag"),
				ID:         strVal(m, "id"),
				Attribute:  strVal(m, "attribute"),
				OldValue:   strVal(m, "oldValue"),
				NewValue:   strVal(m, "newValue"),
				ChangeType: "attribute",
			})
		case "characterData":
			diff.Modified = append(diff.Modified, DOMChange{
				Tag:        "text",
				OldValue:   strVal(m, "oldValue"),
				NewValue:   strVal(m, "newValue"),
				ChangeType: "text",
			})
		}
	}

	diff.HasDiff = len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0
	return diff, nil
}

func parseDOMElement(m map[string]any) DOMElement {
	return DOMElement{
		Tag:     strVal(m, "tag"),
		ID:      strVal(m, "id"),
		Classes: strVal(m, "classes"),
		Text:    strVal(m, "text"),
	}
}
