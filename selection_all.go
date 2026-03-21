package browse

// SelectionAll wraps multiple elements for batch operations.
type SelectionAll struct {
	selections []*Selection
	selector   string
	err        error
}

// Count returns the number of matched elements.
func (sa *SelectionAll) Count() int {
	return len(sa.selections)
}

// Each iterates over each matched element with its index.
func (sa *SelectionAll) Each(fn func(int, *Selection)) error {
	if sa.err != nil {
		return sa.err
	}
	for i, s := range sa.selections {
		fn(i, s)
	}
	return nil
}

// Texts returns the text content of all matched elements.
func (sa *SelectionAll) Texts() ([]string, error) {
	if sa.err != nil {
		return nil, sa.err
	}
	texts := make([]string, 0, len(sa.selections))
	for _, s := range sa.selections {
		t, err := s.Text()
		if err != nil {
			return texts, err
		}
		texts = append(texts, t)
	}
	return texts, nil
}

// First returns the first matched element.
func (sa *SelectionAll) First() *Selection {
	if sa.err != nil || len(sa.selections) == 0 {
		return &Selection{err: &ElementNotFoundError{Selector: sa.selector}}
	}
	return sa.selections[0]
}

// Last returns the last matched element.
func (sa *SelectionAll) Last() *Selection {
	if sa.err != nil || len(sa.selections) == 0 {
		return &Selection{err: &ElementNotFoundError{Selector: sa.selector}}
	}
	return sa.selections[len(sa.selections)-1]
}

// At returns the element at the given index.
func (sa *SelectionAll) At(i int) *Selection {
	if sa.err != nil || i < 0 || i >= len(sa.selections) {
		return &Selection{err: &ElementNotFoundError{Selector: sa.selector}}
	}
	return sa.selections[i]
}

// Filter returns a new SelectionAll containing only elements that pass the predicate.
func (sa *SelectionAll) Filter(fn func(*Selection) bool) *SelectionAll {
	if sa.err != nil {
		return sa
	}
	filtered := make([]*Selection, 0)
	for _, s := range sa.selections {
		if fn(s) {
			filtered = append(filtered, s)
		}
	}
	return &SelectionAll{selections: filtered, selector: sa.selector}
}
