package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	browse "github.com/felixgeelhaar/scout"
)

// DiscoverForm analyzes form fields on the page and returns their labels and selectors.
// If formSelector is empty, discovers all forms on the page.
func (s *Session) DiscoverForm(formSelector string) (*FormDiscoveryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	sel := "document.body"
	if formSelector != "" {
		selJSON, _ := json.Marshal(formSelector)
		sel = fmt.Sprintf("document.querySelector(%s)", selJSON)
	}

	js := fmt.Sprintf(`(function() {
		const root = %s;
		if (!root) return null;

		function findLabel(el) {
			if (el.id) {
				const label = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
				if (label) return label.textContent.trim();
			}
			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return ariaLabel;
			const labelledBy = el.getAttribute('aria-labelledby');
			if (labelledBy) {
				const ref = document.getElementById(labelledBy);
				if (ref) return ref.textContent.trim();
			}
			const parent = el.closest('label');
			if (parent) {
				const clone = parent.cloneNode(true);
				const inputs = clone.querySelectorAll('input,textarea,select');
				inputs.forEach(i => i.remove());
				return clone.textContent.trim();
			}
			if (el.placeholder) return el.placeholder;
			const prev = el.previousElementSibling;
			if (prev && ['LABEL','SPAN','DIV','P','TD','TH'].includes(prev.tagName)) {
				return prev.textContent.trim();
			}
			return el.name || el.id || '';
		}

		function buildSelector(el) {
			if (el.id) return '#' + CSS.escape(el.id);
			if (el.name) return el.tagName.toLowerCase() + '[name="' + el.name + '"]';
			const parent = el.closest('form');
			if (parent) {
				const siblings = parent.querySelectorAll(el.tagName.toLowerCase());
				const idx = Array.from(siblings).indexOf(el);
				if (idx >= 0) return (parent.id ? '#' + CSS.escape(parent.id) + ' ' : 'form ') + el.tagName.toLowerCase() + ':nth-of-type(' + (idx+1) + ')';
			}
			return el.tagName.toLowerCase();
		}

		const fields = [];
		const inputs = root.querySelectorAll('input,textarea,select');
		for (const el of inputs) {
			if (el.type === 'hidden' || el.type === 'submit') continue;
			const field = {
				selector: buildSelector(el),
				label: findLabel(el),
				type: el.type || el.tagName.toLowerCase(),
				name: el.name || '',
				id: el.id || '',
				placeholder: el.placeholder || '',
				required: el.required || false
			};
			if (el.tagName === 'SELECT') {
				field.options = Array.from(el.options).slice(0, 10).map(o => o.text.trim());
			}
			fields.push(field);
		}

		const form = root.tagName === 'FORM' ? root : root.querySelector('form');
		return JSON.stringify({
			formSelector: form ? buildSelector(form) : '',
			action: form ? form.action : '',
			method: form ? form.method : '',
			fields: fields
		});
	})()`, sel)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, fmt.Errorf("agent: form discovery failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("agent: no form found")
	}

	str, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("agent: unexpected form discovery result")
	}

	var discovery FormDiscoveryResult
	if err := json.Unmarshal([]byte(str), &discovery); err != nil {
		return nil, fmt.Errorf("agent: failed to parse form discovery: %w", err)
	}

	return &discovery, nil
}

// FillFormSemantic fills form fields using human-readable names instead of CSS selectors.
// Keys are names like "Email", "Password", "First Name".
// The method auto-discovers form fields and matches by label, name, placeholder, and id.
func (s *Session) FillFormSemantic(fields map[string]string) (*SemanticFillResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	// Discover form fields (unlock not needed since we hold the lock)
	discovery, err := s.discoverFormInternal("")
	if err != nil {
		return nil, err
	}

	result := &SemanticFillResult{
		Fields:  make([]SemanticFieldResult, 0, len(fields)),
		Success: true,
	}

	for humanName, value := range fields {
		best := MatchFormField(humanName, discovery.Fields)
		if best == nil {
			result.Fields = append(result.Fields, SemanticFieldResult{
				HumanName: humanName,
				Error:     fmt.Sprintf("no matching field found for %q", humanName),
			})
			result.Success = false
			continue
		}

		nodeID, err := s.page.QuerySelector(best.Selector)
		if err != nil {
			result.Fields = append(result.Fields, SemanticFieldResult{
				HumanName: humanName,
				Selector:  best.Selector,
				Error:     err.Error(),
			})
			result.Success = false
			continue
		}

		sel := browse.NewSelection(s.page, nodeID, best.Selector)
		if err := sel.Input(value); err != nil {
			result.Fields = append(result.Fields, SemanticFieldResult{
				HumanName: humanName,
				Selector:  best.Selector,
				Error:     err.Error(),
			})
			result.Success = false
			continue
		}

		actual, _ := sel.Value()
		result.Fields = append(result.Fields, SemanticFieldResult{
			HumanName: humanName,
			Selector:  best.Selector,
			Value:     actual,
			Success:   true,
		})
	}

	return result, nil
}

// discoverFormInternal is the non-locking version of DiscoverForm.
func (s *Session) discoverFormInternal(formSelector string) (*FormDiscoveryResult, error) {
	sel := "document.body"
	if formSelector != "" {
		selJSON, _ := json.Marshal(formSelector)
		sel = fmt.Sprintf("document.querySelector(%s)", selJSON)
	}

	// Same JS as DiscoverForm but called without taking the lock
	js := fmt.Sprintf(`(function() {
		const root = %s;
		if (!root) return null;
		function findLabel(el) {
			if (el.id) { const l = document.querySelector('label[for="'+CSS.escape(el.id)+'"]'); if (l) return l.textContent.trim(); }
			if (el.getAttribute('aria-label')) return el.getAttribute('aria-label');
			const lb = el.getAttribute('aria-labelledby'); if (lb) { const r = document.getElementById(lb); if (r) return r.textContent.trim(); }
			const p = el.closest('label'); if (p) { const c = p.cloneNode(true); c.querySelectorAll('input,textarea,select').forEach(i => i.remove()); return c.textContent.trim(); }
			if (el.placeholder) return el.placeholder;
			const prev = el.previousElementSibling; if (prev && ['LABEL','SPAN','DIV','P','TD','TH'].includes(prev.tagName)) return prev.textContent.trim();
			return el.name || el.id || '';
		}
		function buildSelector(el) {
			if (el.id) return '#'+CSS.escape(el.id);
			if (el.name) return el.tagName.toLowerCase()+'[name="'+el.name+'"]';
			return el.tagName.toLowerCase();
		}
		const fields = [];
		for (const el of root.querySelectorAll('input,textarea,select')) {
			if (el.type==='hidden'||el.type==='submit') continue;
			const f = {selector:buildSelector(el),label:findLabel(el),type:el.type||el.tagName.toLowerCase(),name:el.name||'',id:el.id||'',placeholder:el.placeholder||'',required:el.required||false};
			if (el.tagName==='SELECT') f.options=Array.from(el.options).slice(0,10).map(o=>o.text.trim());
			fields.push(f);
		}
		const form = root.tagName==='FORM'?root:root.querySelector('form');
		return JSON.stringify({formSelector:form?buildSelector(form):'',action:form?form.action:'',method:form?form.method:'',fields:fields});
	})()`, sel)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("agent: no form found")
	}
	str, _ := result.(string)
	var discovery FormDiscoveryResult
	if err := json.Unmarshal([]byte(str), &discovery); err != nil {
		return nil, err
	}
	return &discovery, nil
}

// MatchFormField finds the best matching field for a human-readable name using
// weighted fuzzy matching on label, name, id, placeholder, and type.
// Returns nil if no match is found. Exported for direct testing and reuse.
func MatchFormField(humanName string, fields []FormFieldInfo) *FormFieldInfo {
	humanLower := strings.ToLower(humanName)
	var best *FormFieldInfo
	bestScore := 0

	for i := range fields {
		f := &fields[i]
		score := 0

		// Exact label match (highest priority)
		if strings.EqualFold(f.Label, humanName) {
			score = 100
		} else if strings.Contains(strings.ToLower(f.Label), humanLower) {
			score = 80
		}

		// Name/ID match
		if strings.EqualFold(f.Name, humanName) || strings.EqualFold(f.ID, humanName) {
			score = max(score, 90)
		} else if strings.Contains(strings.ToLower(f.Name), humanLower) {
			score = max(score, 70)
		} else if strings.Contains(strings.ToLower(f.ID), humanLower) {
			score = max(score, 60)
		}

		// Placeholder match
		if strings.Contains(strings.ToLower(f.Placeholder), humanLower) {
			score = max(score, 50)
		}

		// Type hint match (e.g., "email" matches type="email")
		if strings.EqualFold(f.Type, humanName) {
			score = max(score, 40)
		}

		if score > bestScore {
			bestScore = score
			best = f
		}
	}

	return best
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
