package engine

import "fmt"

// computeFederal runs the IRS Publication 915 "Worksheet 1" for the taxable
// portion of Social Security benefits. Implementing the full worksheet (rather
// than the simplified "up to 50% / up to 85%" summary) is what makes the numbers
// match a real return.
//
// Worksheet line references are kept in comments for auditability.
func computeFederal(in Input, fr FederalRules) FederalResult {
	base := fr.Base[in.FilingStatus]
	upper := fr.Upper[in.FilingStatus]

	l1 := in.SSBenefits                         // Line 1: total benefits
	l2 := 0.5 * l1                              // Line 2: 50% of benefits
	l3 := in.OtherIncome + in.TaxExemptInterest // Line 3: other income + tax-exempt interest
	provisional := l2 + l3                      // Line 5 (adjustments assumed 0): provisional income

	res := FederalResult{ProvisionalIncome: round2(provisional)}

	// Line 8 = base amount; Line 9 = provisional − base.
	l9 := provisional - base
	if l9 <= 0 {
		res.TaxableAmount = 0
		res.TaxablePercent = 0
		res.Tier = "none"
		res.Explanation = fmt.Sprintf(
			"Provisional income $%.0f is at or below the $%.0f base threshold for %s, so none of the $%.0f in Social Security benefits is federally taxable.",
			provisional, base, filingLabel(in.FilingStatus), l1)
		return res
	}

	secondTier := upper - base // Line 10: width of the 50% band (9000 single / 12000 MFJ / 0 for MFS-together)
	l10 := secondTier
	l11 := l9 - l10 // amount into the 85% band
	if l11 < 0 {
		l11 = 0
	}
	l12 := min(l9, l10) // Line 12: amount in the 50% band
	l13 := 0.5 * l12    // Line 13
	l14 := min(l13, l2) // Line 14: 50%-band contribution, capped at 50% of benefits
	l15 := 0.85 * l11   // Line 15: 85%-band contribution
	l16 := l14 + l15    // Line 16
	l17 := 0.85 * l1    // Line 17: hard cap of 85% of benefits
	taxable := min(l16, l17)

	res.TaxableAmount = round2(taxable)
	if l1 > 0 {
		res.TaxablePercent = round2(taxable / l1 * 100)
	}

	// With no benefits (or none taxable after the 85% cap), report "none"
	// regardless of where provisional income landed.
	if res.TaxableAmount <= 0 {
		res.Tier = "none"
		res.Explanation = fmt.Sprintf(
			"None of the $%.0f in Social Security benefits is federally taxable.", l1)
		return res
	}

	if l11 > 0 {
		res.Tier = "up_to_85"
		res.Explanation = fmt.Sprintf(
			"Provisional income $%.0f exceeds the $%.0f upper threshold for %s, so up to 85%% of benefits is taxable. $%.2f of the $%.0f benefit (%.1f%%) is federally taxable.",
			provisional, upper, filingLabel(in.FilingStatus), res.TaxableAmount, l1, res.TaxablePercent)
	} else {
		res.Tier = "up_to_50"
		res.Explanation = fmt.Sprintf(
			"Provisional income $%.0f is between the $%.0f base and $%.0f upper thresholds for %s, so up to 50%% of benefits is taxable. $%.2f of the $%.0f benefit (%.1f%%) is federally taxable.",
			provisional, base, upper, filingLabel(in.FilingStatus), res.TaxableAmount, l1, res.TaxablePercent)
	}
	return res
}

func filingLabel(f FilingStatus) string {
	switch f {
	case Single:
		return "a single filer"
	case HoH:
		return "a head-of-household filer"
	case MFJ:
		return "married filing jointly"
	case MFSApart:
		return "married filing separately (lived apart)"
	case MFSTogether:
		return "married filing separately (lived together)"
	default:
		return string(f)
	}
}
