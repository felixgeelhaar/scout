package agent

import "fmt"

// ContentOptions controls how page content is extracted and truncated.
type ContentOptions struct {
	// MaxLength is the maximum character length of returned content. Default 4000.
	MaxLength int
	// MaxLinks caps the number of links returned in observations. Default 25.
	MaxLinks int
	// MaxInputs caps the number of inputs returned. Default 20.
	MaxInputs int
	// MaxButtons caps the number of buttons returned. Default 15.
	MaxButtons int
	// MaxItems caps ExtractAll results. Default 50.
	MaxItems int
	// MaxRows caps table rows. Default 100.
	MaxRows int
	// MaxScreenshotBytes is the maximum screenshot size in bytes. Default 5MB.
	// Screenshots exceeding this are auto-compressed (JPEG + downscale).
	MaxScreenshotBytes int
}

// DefaultContentOptions returns sensible defaults for LLM context windows.
func DefaultContentOptions() ContentOptions {
	return ContentOptions{
		MaxLength:          4000,
		MaxLinks:           25,
		MaxInputs:          20,
		MaxButtons:         15,
		MaxItems:           50,
		MaxRows:            100,
		MaxScreenshotBytes: 5 * 1024 * 1024, // 5MB
	}
}

// SetContentOptions configures content limits for the session.
func (s *Session) SetContentOptions(opts ContentOptions) {
	s.contentOpts = opts
}

// Markdown returns a compact markdown representation of the page content.
// This is much smaller than raw HTML and easier for LLMs to process.
func (s *Session) Markdown() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return "", err
	}

	maxLen := s.contentOpts.MaxLength
	if maxLen == 0 {
		maxLen = 4000
	}

	js := fmt.Sprintf(`(function() {
		function toMd(el, depth) {
			if (!el) return '';
			const tag = el.tagName ? el.tagName.toLowerCase() : '';
			const skip = ['script','style','noscript','svg','path','meta','link','head'];
			if (skip.includes(tag)) return '';

			if (el.nodeType === 3) return el.textContent;

			let md = '';
			const children = Array.from(el.childNodes).map(c => toMd(c, depth+1)).join('');

			switch(tag) {
				case 'h1': md = '\n# ' + children.trim() + '\n'; break;
				case 'h2': md = '\n## ' + children.trim() + '\n'; break;
				case 'h3': md = '\n### ' + children.trim() + '\n'; break;
				case 'h4': case 'h5': case 'h6':
					md = '\n#### ' + children.trim() + '\n'; break;
				case 'p': md = '\n' + children.trim() + '\n'; break;
				case 'br': md = '\n'; break;
				case 'a':
					const href = el.getAttribute('href') || '';
					const text = children.trim();
					md = text ? '[' + text + '](' + href + ')' : '';
					break;
				case 'img':
					const alt = el.getAttribute('alt') || '';
					md = alt ? '![' + alt + ']' : '';
					break;
				case 'strong': case 'b': md = '**' + children.trim() + '**'; break;
				case 'em': case 'i': md = '*' + children.trim() + '*'; break;
				case 'code': md = '`+"`"+`' + children.trim() + '`+"`"+`'; break;
				case 'pre': md = '\n`+"```"+`\n' + children.trim() + '\n`+"```"+`\n'; break;
				case 'li': md = '\n- ' + children.trim(); break;
				case 'ul': case 'ol': md = '\n' + children + '\n'; break;
				case 'tr': md = '| ' + children; break;
				case 'th': md = children.trim() + ' | '; break;
				case 'td': md = children.trim() + ' | '; break;
				case 'thead':
					md = children + '\n' + '|---'.repeat(el.querySelectorAll('th').length) + '|';
					break;
				case 'table': md = '\n' + children + '\n'; break;
				case 'input':
					const t = el.type || 'text';
					const n = el.name || el.id || '';
					const v = el.value || '';
					const ph = el.placeholder || '';
					md = '[input:' + t + ' name=' + n;
					if (v) md += ' value="' + v + '"';
					if (ph) md += ' placeholder="' + ph + '"';
					md += ']';
					break;
				case 'button':
					md = '[button: ' + children.trim() + ']';
					break;
				case 'select':
					const opts = Array.from(el.options).slice(0, 5).map(o => o.text).join(', ');
					md = '[select name=' + (el.name||el.id||'') + ': ' + opts + ']';
					break;
				case 'textarea':
					md = '[textarea name=' + (el.name||el.id||'') + ']';
					break;
				case 'form':
					md = '\n---form---\n' + children + '\n---/form---\n';
					break;
				case 'nav':
					md = '\n[nav] ' + children + '\n';
					break;
				default: md = children; break;
			}
			return md;
		}
		let result = toMd(document.body, 0);
		// Collapse whitespace
		result = result.replace(/\n{3,}/g, '\n\n').trim();
		return result.slice(0, %d);
	})()`, maxLen)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return "", fmt.Errorf("agent: markdown conversion failed: %w", err)
	}
	md, _ := result.(string)
	return md, nil
}

// ReadableText extracts just the main readable text content, stripping navigation,
// sidebars, and boilerplate. Uses a heuristic based on text density.
func (s *Session) ReadableText() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return "", err
	}

	maxLen := s.contentOpts.MaxLength
	if maxLen == 0 {
		maxLen = 4000
	}

	js := fmt.Sprintf(`(function() {
		// Simple readability: find the element with the most text content
		// that isn't a container of the entire page
		const candidates = document.querySelectorAll('article, main, [role="main"], .content, .post, .article, .entry, #content, #main');
		let best = null;
		let bestLen = 0;

		for (const el of candidates) {
			const text = el.innerText || '';
			if (text.length > bestLen) {
				best = el;
				bestLen = text.length;
			}
		}

		// Fallback: use body but skip nav, header, footer, aside
		if (!best || bestLen < 100) {
			const clone = document.body.cloneNode(true);
			for (const tag of ['nav', 'header', 'footer', 'aside', 'script', 'style', 'noscript']) {
				for (const el of clone.querySelectorAll(tag)) el.remove();
			}
			return clone.innerText.replace(/\n{3,}/g, '\n\n').trim().slice(0, %d);
		}

		return best.innerText.replace(/\n{3,}/g, '\n\n').trim().slice(0, %d);
	})()`, maxLen, maxLen)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return "", fmt.Errorf("agent: readability extraction failed: %w", err)
	}
	text, _ := result.(string)
	return text, nil
}

// AccessibilityTree returns a compact tree representation of the page's
// accessible elements. This is much smaller than HTML and captures the
// semantic structure that matters for interaction.
func (s *Session) AccessibilityTree() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensurePage(); err != nil {
		return "", err
	}

	maxLen := s.contentOpts.MaxLength
	if maxLen == 0 {
		maxLen = 8000
	}

	return s.accessibilityTreeJS(maxLen)
}

func (s *Session) accessibilityTreeJS(maxLen int) (string, error) {
	js := fmt.Sprintf(`(function() {
		const lines = [];
		function walk(el, depth) {
			if (lines.join('\\n').length > %d) return;
			if (!el || !el.tagName) return;
			const tag = el.tagName.toLowerCase();
			const skip = ['script','style','noscript','svg','path','br','hr'];
			if (skip.includes(tag)) return;

			const indent = '  '.repeat(depth);
			const role = el.getAttribute('role') || '';
			const ariaLabel = el.getAttribute('aria-label') || '';
			const id = el.id ? '#' + el.id : '';
			const name = el.name ? '[name=' + el.name + ']' : '';

			// Only emit meaningful nodes
			let line = '';
			switch(tag) {
				case 'a':
					const href = el.getAttribute('href') || '';
					const text = el.textContent.trim().slice(0, 80);
					if (text) line = indent + 'link "' + text + '" -> ' + href;
					break;
				case 'button':
					line = indent + 'button "' + el.textContent.trim().slice(0, 50) + '"' + id;
					break;
				case 'input':
					const t = el.type || 'text';
					const ph = el.placeholder ? ' placeholder="' + el.placeholder + '"' : '';
					const val = el.value ? ' value="' + el.value + '"' : '';
					line = indent + 'input[' + t + ']' + id + name + ph + val;
					break;
				case 'textarea':
					line = indent + 'textarea' + id + name;
					break;
				case 'select':
					const opts = Array.from(el.options).slice(0, 3).map(o => o.text).join(', ');
					line = indent + 'select' + id + name + ' (' + opts + ')';
					break;
				case 'img':
					const alt = el.getAttribute('alt');
					if (alt) line = indent + 'img "' + alt + '"';
					break;
				case 'h1': case 'h2': case 'h3': case 'h4':
					line = indent + tag + ' "' + el.textContent.trim().slice(0, 100) + '"';
					break;
				case 'label':
					const forAttr = el.getAttribute('for') || '';
					line = indent + 'label "' + el.textContent.trim().slice(0, 50) + '"' + (forAttr ? ' for=' + forAttr : '');
					break;
				case 'form':
					line = indent + 'form' + id;
					break;
				case 'nav':
					line = indent + 'nav' + (ariaLabel ? ' "' + ariaLabel + '"' : '');
					break;
				case 'main': case 'article': case 'section': case 'aside': case 'header': case 'footer':
					line = indent + tag + (ariaLabel ? ' "' + ariaLabel + '"' : '') + id;
					break;
				default:
					if (role) line = indent + '[role=' + role + ']' + id;
					else if (el.onclick || el.getAttribute('tabindex')) {
						const text = el.textContent.trim().slice(0, 50);
						if (text) line = indent + 'interactive "' + text + '"' + id;
					}
					break;
			}

			if (line) lines.push(line);

			for (const child of el.children) {
				walk(child, line ? depth + 1 : depth);
			}
		}
		walk(document.body, 0);
		return lines.join('\\n');
	})()`, maxLen)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return "", err
	}
	tree, _ := result.(string)
	return tree, nil
}
