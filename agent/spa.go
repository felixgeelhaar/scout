package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// DetectedFrameworks returns which frontend frameworks are active on the current page.
func (s *Session) DetectedFrameworks() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	js := `(function() {
		const d = [];
		try {
		// Fast checks (globals only, no DOM scanning)
		if (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) d.push('react');
		if (window.Vue) d.push('vue2');
		if (window.__VUE__ || document.querySelector('[data-v-app]')) d.push('vue3');
		if (window.ng || window.getAllAngularTestabilities || document.querySelector('[ng-version]')) d.push('angular');
		if (window._$HY || window.$SOLID_DEVTOOLS) d.push('solid');
		if (window.preact) d.push('preact');
		if (window.Alpine || document.querySelector('[x-data]')) d.push('alpine');
		if (window.htmx) d.push('htmx');
		if (window.Stimulus || document.querySelector('[data-controller]')) d.push('stimulus');
		if (window.Ember || window.Em) d.push('ember');
		if (window.__QWIK_MANIFEST__) d.push('qwik');
		if (window.__NEXT_DATA__ || document.getElementById('__NEXT_DATA__')) d.push('nextjs');
		if (window.__NUXT__ || document.getElementById('__NUXT_DATA__')) d.push('nuxt');
		if (window.__remixContext || document.getElementById('__remixContext')) d.push('remix');
		if (document.getElementById('__sveltekit_data')) d.push('sveltekit');
		if (document.getElementById('___gatsby') || window.___GATSBY_INTERNAL_PLUGINS) d.push('gatsby');
		if (document.querySelector('[data-astro-island]') || window.__ASTRO__) d.push('astro');
		// Slower checks — scan a sample of elements for framework markers
		const sample = document.querySelectorAll('#root, #app, #__next, [data-reactroot], body > div');
		for (const el of sample) {
			const keys = Object.keys(el);
			if (!d.includes('react') && keys.some(k => k.startsWith('__reactFiber'))) d.push('react');
			if (!d.includes('vue2') && el.__vue__) d.push('vue2');
			if (!d.includes('vue3') && (el.__vueParentComponent || el.__vue_app__)) d.push('vue3');
			if (!d.includes('svelte') && (el.__svelte_meta || el.$$)) d.push('svelte');
			if (!d.includes('preact') && keys.some(k => k.startsWith('__preact'))) d.push('preact');
			if (!d.includes('lit') && el.shadowRoot && el.renderRoot) d.push('lit');
		}
		} catch(e) {}
		return JSON.stringify(d);
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	str, _ := result.(string)
	var frameworks []string
	_ = json.Unmarshal([]byte(str), &frameworks)
	return frameworks, nil
}

// ComponentState extracts state/props from a framework component at the given selector.
// Auto-detects the framework (React, Vue 2/3, Svelte, Preact, Angular, Alpine, Lit).
func (s *Session) ComponentState(selector string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	selectorJSON, _ := json.Marshal(selector)
	js := fmt.Sprintf(`(function() {
		const el = document.querySelector(%s);
		if (!el) return null;
		const result = {framework: null};

		// React (fiber)
		const fiberKey = Object.keys(el).find(k => k.startsWith('__reactFiber'));
		if (fiberKey) {
			result.framework = 'react';
			const fiber = el[fiberKey];
			let current = fiber;
			while (current) {
				if (current.memoizedProps || current.memoizedState) {
					result.props = current.memoizedProps;
					const states = [];
					let hook = current.memoizedState;
					while (hook) {
						if (hook.memoizedState !== undefined && typeof hook.memoizedState !== 'function') {
							try { states.push(JSON.parse(JSON.stringify(hook.memoizedState))); } catch(e) {}
						}
						hook = hook.next;
					}
					if (states.length > 0) result.state = states;
					break;
				}
				current = current.return;
			}
			return JSON.stringify(result);
		}

		// Vue 2
		if (el.__vue__) {
			result.framework = 'vue2';
			try { result.data = JSON.parse(JSON.stringify(el.__vue__.$data || {})); } catch(e) {}
			result.props = el.__vue__.$props || {};
			return JSON.stringify(result);
		}

		// Vue 3
		if (el.__vueParentComponent) {
			result.framework = 'vue3';
			const inst = el.__vueParentComponent;
			if (inst.setupState) {
				const data = {};
				for (const k of Object.keys(inst.setupState)) {
					try { if (typeof inst.setupState[k] !== 'function') data[k] = JSON.parse(JSON.stringify(inst.setupState[k])); } catch(e) {}
				}
				result.data = data;
			}
			result.props = inst.props || {};
			return JSON.stringify(result);
		}

		// Svelte
		if (el.$$) {
			result.framework = 'svelte';
			result.ctx = el.$$.ctx;
			return JSON.stringify(result);
		}

		// Preact
		const preactKey = Object.keys(el).find(k => k.startsWith('__preact'));
		if (preactKey) {
			result.framework = 'preact';
			const f = el[preactKey];
			result.props = f.props;
			if (f._component) result.state = f._component.state;
			return JSON.stringify(result);
		}

		// Angular (Ivy)
		if (window.ng && window.ng.getComponent) {
			try {
				const comp = window.ng.getComponent(el);
				if (comp) {
					result.framework = 'angular';
					const props = {};
					for (const k of Object.getOwnPropertyNames(comp)) {
						try { if (typeof comp[k] !== 'function') props[k] = JSON.parse(JSON.stringify(comp[k])); } catch(e) {}
					}
					result.state = props;
					return JSON.stringify(result);
				}
			} catch(e) {}
		}

		// Alpine.js
		if (el._x_dataStack || el.__x) {
			result.framework = 'alpine';
			result.data = el._x_dataStack ? el._x_dataStack[0] : (el.__x ? el.__x.$data : {});
			return JSON.stringify(result);
		}

		// Lit / Web Components
		if (el.shadowRoot && el.constructor.properties) {
			result.framework = 'lit';
			const props = {};
			for (const [k] of el.constructor.properties) {
				try { props[k] = el[k]; } catch(e) {}
			}
			result.properties = props;
			return JSON.stringify(result);
		}

		return null;
	})()`, selectorJSON)

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("no framework component found at %s", selector)
	}
	str, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(str), &state); err != nil {
		return nil, err
	}
	return state, nil
}

// ReactState extracts React component state/props from an element.
func (s *Session) ReactState(selector string) (map[string]any, error) {
	return s.ComponentState(selector)
}

// VueState extracts Vue component data from an element.
func (s *Session) VueState(selector string) (map[string]any, error) {
	return s.ComponentState(selector)
}

// GetAppState extracts global application state from all detected frameworks.
// Checks: Redux, Next.js, Nuxt, Remix, SvelteKit, Gatsby, Alpine stores,
// HTMX config, and common SSR hydration patterns.
func (s *Session) GetAppState() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	js := `(function() {
		const state = {};

		// Next.js
		try {
			const nd = document.getElementById('__NEXT_DATA__');
			if (nd) {
				const d = JSON.parse(nd.textContent);
				state.nextjs = {page: d.page, buildId: d.buildId, props: d.props?.pageProps};
			} else if (window.__NEXT_DATA__) {
				state.nextjs = {page: window.__NEXT_DATA__.page, props: window.__NEXT_DATA__.props?.pageProps};
			}
		} catch(e) {}

		// Nuxt
		try {
			if (window.__NUXT__) state.nuxt = window.__NUXT__.data || window.__NUXT__.state || window.__NUXT__;
			const nd = document.getElementById('__NUXT_DATA__');
			if (nd) state.nuxt = JSON.parse(nd.textContent);
		} catch(e) {}

		// Remix
		try {
			if (window.__remixContext) state.remix = window.__remixContext;
			const rc = document.getElementById('__remixContext');
			if (rc) state.remix = JSON.parse(rc.textContent);
		} catch(e) {}

		// SvelteKit
		try {
			const sk = document.getElementById('__sveltekit_data');
			if (sk) state.sveltekit = JSON.parse(sk.textContent);
		} catch(e) {}

		// Gatsby
		try {
			const g = document.querySelector('script[id="gatsby-chunk-mapping"]');
			if (g) state.gatsby = JSON.parse(g.textContent);
		} catch(e) {}

		// Astro islands
		try {
			const islands = document.querySelectorAll('[data-astro-island]');
			if (islands.length > 0) {
				state.astro = Array.from(islands).map(i => ({
					component: i.getAttribute('data-astro-island'),
					props: i.dataset.astroProps ? JSON.parse(decodeURIComponent(i.dataset.astroProps)) : {}
				}));
			}
		} catch(e) {}

		// Alpine stores
		try {
			if (window.Alpine && window.Alpine._stores) {
				state.alpine = {};
				for (const [k, v] of Object.entries(window.Alpine._stores)) {
					try { state.alpine[k] = JSON.parse(JSON.stringify(v)); } catch(e) {}
				}
			}
		} catch(e) {}

		// HTMX
		try {
			if (window.htmx) state.htmx = {version: window.htmx.version, config: window.htmx.config};
		} catch(e) {}

		// Qwik
		try {
			if (window.__QWIK_MANIFEST__) state.qwik = {manifest: true};
		} catch(e) {}

		// Generic SSR hydration state
		const hydrationKeys = ['__INITIAL_STATE__','__APP_STATE__','__PRELOADED_STATE__','__APP_INITIAL_STATE__','__INITIAL_DATA__'];
		for (const k of hydrationKeys) {
			try { if (window[k]) state[k] = window[k]; } catch(e) {}
		}

		// JSON script tags (common hydration pattern)
		const jsonScripts = document.querySelectorAll('script[type="application/json"]');
		if (jsonScripts.length > 0) {
			state._hydrationScripts = Array.from(jsonScripts).slice(0, 5).map(s => {
				try { return {id: s.id, data: JSON.parse(s.textContent)}; } catch(e) { return {id: s.id}; }
			});
		}

		if (Object.keys(state).length === 0) return null;
		return JSON.stringify(state);
	})()`

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	str, ok := result.(string)
	if !ok {
		return nil, nil
	}
	var appState map[string]any
	if err := json.Unmarshal([]byte(str), &appState); err != nil {
		return nil, err
	}
	return appState, nil
}

// WaitForSPA waits for SPA framework hydration/rendering to complete.
// Detects React, Vue, Angular, Svelte, Next.js, Nuxt, and generic content presence.
func (s *Session) WaitForSPA() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return err
	}

	js := `new Promise(resolve => {
		function check() {
			const ready =
				(document.getElementById('root') && document.getElementById('root').children.length > 0) ||
				(document.getElementById('app') && document.getElementById('app').children.length > 0) ||
				(document.getElementById('__next') && document.getElementById('__next').children.length > 0) ||
				(document.getElementById('__nuxt') && document.getElementById('__nuxt').children.length > 0) ||
				document.querySelector('[data-v-app]') !== null ||
				document.querySelector('[ng-version]') !== null ||
				document.querySelector('[data-astro-island]') !== null ||
				document.querySelector('[data-sveltekit-router]') !== null ||
				(document.body && document.body.innerText.trim().length > 100);
			if (ready) resolve(true);
			else requestAnimationFrame(check);
		}
		if (document.readyState === 'complete') setTimeout(check, 100);
		else window.addEventListener('load', () => setTimeout(check, 100));
		setTimeout(() => resolve(true), 10000);
	})`

	_, err := s.page.Evaluate(js)
	return err
}

// DispatchEvent dispatches a DOM event on an element.
func (s *Session) DispatchEvent(selector, eventType string, detail map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return err
	}

	selectorJSON, _ := json.Marshal(selector)
	detailJSON, _ := json.Marshal(detail)

	js := fmt.Sprintf(`(function() {
		const el = document.querySelector(%s);
		if (!el) return false;
		el.dispatchEvent(new CustomEvent(%q, {detail: %s, bubbles: true, cancelable: true}));
		return true;
	})()`, selectorJSON, eventType, string(detailJSON))

	result, err := s.page.Evaluate(js)
	if err != nil {
		return err
	}
	if b, ok := result.(bool); !ok || !b {
		return fmt.Errorf("element %s not found", selector)
	}
	return nil
}

// WaitForRouteChange waits for a SPA client-side route change (pushState/replaceState/hashchange).
func (s *Session) WaitForRouteChange(timeout time.Duration) (*PageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = s.timeout
	}

	js := fmt.Sprintf(`new Promise((resolve, reject) => {
		const timer = setTimeout(() => reject(new Error('timeout waiting for route change')), %d);
		const origPush = history.pushState;
		const origReplace = history.replaceState;
		function done() {
			clearTimeout(timer);
			window.removeEventListener('popstate', done);
			window.removeEventListener('hashchange', done);
			history.pushState = origPush;
			history.replaceState = origReplace;
			resolve(window.location.href);
		}
		window.addEventListener('popstate', done);
		window.addEventListener('hashchange', done);
		history.pushState = function() { origPush.apply(this, arguments); done(); };
		history.replaceState = function() { origReplace.apply(this, arguments); done(); };
	})`, timeout.Milliseconds())

	result, err := s.page.Evaluate(js)
	if err != nil {
		return nil, err
	}
	urlStr, _ := result.(string)
	title, _ := s.page.Evaluate(`document.title`)
	titleStr, _ := title.(string)
	return &PageResult{URL: urlStr, Title: titleStr}, nil
}
