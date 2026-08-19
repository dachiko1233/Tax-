package engine

import "strings"

// Engine runs computations against a versioned rule Provider.
type Engine struct {
	provider Provider
}

// New builds an Engine from a rule provider. Pass DefaultProvider() for the
// built-in rulesets, or a DB-backed provider in production.
func New(p Provider) *Engine {
	return &Engine{provider: p}
}

// Calculate runs the federal worksheet then the selected state's rule and
// assembles the combined result with a flat list of explanation strings.
func (e *Engine) Calculate(in Input) (*Result, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	rs, err := e.provider.ForYear(in.TaxYear)
	if err != nil {
		return nil, err
	}

	fed := computeFederal(in, rs.Federal)
	st := computeState(in, rs, fed.TaxableAmount)

	out := &Result{
		TaxYear:      in.TaxYear,
		Federal:      fed,
		State:        st,
		CombinedTax:  round2(st.EstimatedTax),
		Explanations: []string{fed.Explanation},
	}
	if st.Explanation != "" {
		out.Explanations = append(out.Explanations, st.Explanation)
	}
	return out, nil
}

// normalizeState upper-cases and trims a state code.
func normalizeState(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
