package agent

// ObserveWithBudget returns a page observation constrained to approximately
// the given token budget. Content is prioritized: interactive elements first,
// then headings, then main content. 1 token ≈ 4 characters.
func (s *Session) ObserveWithBudget(budget int) (*Observation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensurePage(); err != nil {
		return nil, err
	}

	// Temporarily adjust content options for this observation
	savedOpts := s.contentOpts
	defer func() { s.contentOpts = savedOpts }()

	charBudget := budget * 4
	s.contentOpts.MaxLength = min(charBudget, s.contentOpts.MaxLength)

	// Allocate budget: 40% interactive, 10% headings, 35% text, 15% misc
	interactiveBudget := budget * 40 / 100
	textBudget := budget * 35 / 100

	// Cap interactive elements based on budget (~30 tokens per element)
	tokensPerElement := 30
	s.contentOpts.MaxLinks = min(interactiveBudget/tokensPerElement, s.contentOpts.MaxLinks)
	s.contentOpts.MaxInputs = min(interactiveBudget/(tokensPerElement*2), s.contentOpts.MaxInputs)
	s.contentOpts.MaxButtons = min(interactiveBudget/(tokensPerElement*2), s.contentOpts.MaxButtons)

	obs, err := s.observeInternal()
	if err != nil {
		return nil, err
	}

	// Truncate text to remaining budget
	if len(obs.Text) > textBudget*4 {
		obs.Text = obs.Text[:textBudget*4]
	}

	return obs, nil
}

// EstimateTokens returns an approximate token count for a string.
// Uses the heuristic of 1 token ≈ 4 characters.
func EstimateTokens(s string) int {
	return (len(s) + 3) / 4
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
