package engine

import "fmt"

// computeState applies the ruleset for in.State. Non-taxing states (and "")
// return a non-taxable result. federallyTaxable is the federal worksheet output,
// which every state here uses as the starting base.
func computeState(in Input, rs *RuleSet, federallyTaxable float64) StateResult {
	code := normalizeState(in.State)
	if code == "" {
		return StateResult{State: "", Taxable: false,
			Explanation: "No state selected; state Social Security tax not computed."}
	}
	sr, ok := rs.States[code]
	if !ok {
		return StateResult{State: code, Taxable: false,
			Explanation: fmt.Sprintf("%s does not tax Social Security benefits.", code)}
	}

	agi := in.agi(federallyTaxable)
	switch sr.Kind {
	case kindColorado:
		return stateColorado(in, sr, federallyTaxable)
	case kindMontana:
		return stateMontana(in, sr, federallyTaxable)
	case kindFRAExempt:
		return stateFRAExempt(in, sr, federallyTaxable, agi)
	case kindExemptBelowCap:
		return stateExemptBelowCap(in, sr, federallyTaxable, agi)
	default:
		return StateResult{State: code, Taxable: false,
			Explanation: fmt.Sprintf("Unknown rule kind for %s.", code)}
	}
}

// taxableFraction returns the portion (0..1) of benefits that becomes taxable
// given AGI relative to the cap and phase-in width. Below cap => 0; above
// cap+width (or width 0) => 1; linear in between.
func taxableFraction(agi, cap, width float64) float64 {
	if agi <= cap {
		return 0
	}
	if width <= 0 || agi >= cap+width {
		return 1
	}
	return (agi - cap) / width
}

func stateExemptBelowCap(in Input, sr *StateRules, federallyTaxable, agi float64) StateResult {
	cap := sr.cap(in.FilingStatus)
	frac := taxableFraction(agi, cap, sr.PhaseOutWidth)
	taxable := round2(federallyTaxable * frac)
	tax := round2(taxable * sr.FlatRate)

	res := StateResult{State: sr.Code, Taxable: true,
		TaxableAmount: taxable, Rate: sr.FlatRate, EstimatedTax: tax}
	switch {
	case frac == 0:
		res.Explanation = fmt.Sprintf(
			"%s fully exempts Social Security because AGI $%.0f is at or below the $%.0f cap for this filing status.",
			sr.Name, agi, cap)
	case frac < 1:
		res.Explanation = fmt.Sprintf(
			"%s partially taxes Social Security: AGI $%.0f is $%.0f into the $%.0f phase-in band above the $%.0f cap, so %.0f%% ($%.2f) of the federally taxable benefit is taxed at %.2f%% ≈ $%.2f.",
			sr.Name, agi, agi-cap, sr.PhaseOutWidth, cap, frac*100, taxable, sr.FlatRate*100, tax)
	default:
		res.Explanation = fmt.Sprintf(
			"%s taxes the full federally taxable benefit ($%.2f) at %.2f%% ≈ $%.2f because AGI $%.0f exceeds the exemption cap.",
			sr.Name, taxable, sr.FlatRate*100, tax, agi)
	}
	return res
}

func stateFRAExempt(in Input, sr *StateRules, federallyTaxable, agi float64) StateResult {
	// Exemption requires BOTH full retirement age AND AGI at/below the cap.
	if in.AtFRA {
		return stateExemptBelowCap(in, sr, federallyTaxable, agi)
	}
	taxable := round2(federallyTaxable)
	tax := round2(taxable * sr.FlatRate)
	return StateResult{State: sr.Code, Taxable: true,
		TaxableAmount: taxable, Rate: sr.FlatRate, EstimatedTax: tax,
		Explanation: fmt.Sprintf(
			"%s exempts Social Security only at full retirement age; this taxpayer is not at FRA, so the full federally taxable benefit ($%.2f) is taxed at %.2f%% ≈ $%.2f.",
			sr.Name, taxable, sr.FlatRate*100, tax)}
}

func stateColorado(in Input, sr *StateRules, federallyTaxable float64) StateResult {
	if in.Age >= sr.AgeThreshold {
		return StateResult{State: sr.Code, Taxable: true,
			TaxableAmount: 0, Rate: sr.FlatRate, EstimatedTax: 0,
			Explanation: fmt.Sprintf(
				"Colorado lets taxpayers age %d+ subtract all federally taxed Social Security, so $0 is taxable at the state level.",
				sr.AgeThreshold)}
	}
	taxable := round2(federallyTaxable)
	tax := round2(taxable * sr.FlatRate)
	return StateResult{State: sr.Code, Taxable: true,
		TaxableAmount: taxable, Rate: sr.FlatRate, EstimatedTax: tax,
		Explanation: fmt.Sprintf(
			"Colorado taxes the federally taxable benefit ($%.2f) at the flat %.2f%% rate ≈ $%.2f for taxpayers under %d.",
			taxable, sr.FlatRate*100, tax, sr.AgeThreshold)}
}

func stateMontana(in Input, sr *StateRules, federallyTaxable float64) StateResult {
	base := federallyTaxable
	subApplied := 0.0
	if in.Age >= sr.AgeThreshold {
		subApplied = min(sr.Subtraction, base)
		base -= subApplied
	}
	taxable := round2(base)
	tax := round2(taxable * sr.FlatRate)
	res := StateResult{State: sr.Code, Taxable: true,
		TaxableAmount: taxable, Rate: sr.FlatRate, EstimatedTax: tax}
	if subApplied > 0 {
		res.Explanation = fmt.Sprintf(
			"Montana follows federal inclusion ($%.2f) and applies a $%.0f subtraction for age %d+, leaving $%.2f taxed at %.2f%% ≈ $%.2f.",
			round2(federallyTaxable), subApplied, sr.AgeThreshold, taxable, sr.FlatRate*100, tax)
	} else {
		res.Explanation = fmt.Sprintf(
			"Montana follows federal inclusion: $%.2f is taxed at %.2f%% ≈ $%.2f (no age subtraction under %d).",
			taxable, sr.FlatRate*100, tax, sr.AgeThreshold)
	}
	return res
}
