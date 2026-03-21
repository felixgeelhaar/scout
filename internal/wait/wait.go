// Package wait provides context-aware auto-wait utilities for page readiness.
package wait

import (
	"context"
	"fmt"
	"time"
)

// Evaluator can execute JavaScript expressions (matches Page.Evaluate).
type Evaluator interface {
	Evaluate(expression string) (any, error)
}

// ForLoad waits until document.readyState is "complete".
func ForLoad(ctx context.Context, eval Evaluator) error {
	return poll(ctx, func() bool {
		result, err := eval.Evaluate(`document.readyState`)
		if err != nil {
			return false
		}
		s, ok := result.(string)
		return ok && s == "complete"
	}, "page load")
}

// ForSelector waits until at least one element matches the CSS selector.
func ForSelector(ctx context.Context, eval Evaluator, selector string) error {
	js := fmt.Sprintf(`document.querySelector(%q) !== null`, selector)
	return poll(ctx, func() bool {
		result, err := eval.Evaluate(js)
		if err != nil {
			return false
		}
		b, ok := result.(bool)
		return ok && b
	}, fmt.Sprintf("selector %q", selector))
}

// ForVisible waits until the element matching selector is visible.
func ForVisible(ctx context.Context, eval Evaluator, selector string) error {
	js := fmt.Sprintf(`(function() {
		const el = document.querySelector(%q);
		if (!el) return false;
		const style = window.getComputedStyle(el);
		return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
	})()`, selector)

	return poll(ctx, func() bool {
		result, err := eval.Evaluate(js)
		if err != nil {
			return false
		}
		b, ok := result.(bool)
		return ok && b
	}, fmt.Sprintf("%q visible", selector))
}

// ForHidden waits until the element matching selector is hidden or absent.
func ForHidden(ctx context.Context, eval Evaluator, selector string) error {
	js := fmt.Sprintf(`(function() {
		const el = document.querySelector(%q);
		if (!el) return true;
		const style = window.getComputedStyle(el);
		return style.display === 'none' || style.visibility === 'hidden';
	})()`, selector)

	return poll(ctx, func() bool {
		result, err := eval.Evaluate(js)
		if err != nil {
			return false
		}
		b, ok := result.(bool)
		return ok && b
	}, fmt.Sprintf("%q hidden", selector))
}

// ForFunction waits until a JavaScript expression returns true.
func ForFunction(ctx context.Context, eval Evaluator, js string) error {
	return poll(ctx, func() bool {
		result, err := eval.Evaluate(js)
		if err != nil {
			return false
		}
		b, ok := result.(bool)
		return ok && b
	}, "condition")
}

func poll(ctx context.Context, check func() bool, desc string) error {
	for {
		if check() {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait: timeout waiting for %s", desc)
		case <-time.After(50 * time.Millisecond):
		}
	}
}
