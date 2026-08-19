package engine

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestComputeFederal(t *testing.T) {
	fr := rules2026().Federal

	cases := []struct {
		name        string
		in          Input
		wantTaxable float64
		wantTier    string
	}{
		{"single below base", Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 10000}, 0, "none"},
		{"single at base exact", Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 15000}, 0, "none"}, // PI=25000 == base
		{"single 50% band", Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 20000}, 2500, "up_to_50"},
		{"single 85% band capped", Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 40000}, 17000, "up_to_85"},
		{"municipal interest pushes into 85%", Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 20000, TaxExemptInterest: 5000}, 5350, "up_to_85"},
		{"mfj 85% band", Input{FilingStatus: MFJ, SSBenefits: 30000, OtherIncome: 30000}, 6850, "up_to_85"},
		{"mfj below base", Input{FilingStatus: MFJ, SSBenefits: 20000, OtherIncome: 20000}, 0, "none"}, // PI=30000 <= 32000
		{"mfs together always 85%", Input{FilingStatus: MFSTogether, SSBenefits: 20000, OtherIncome: 10000}, 17000, "up_to_85"},
		{"zero benefits", Input{FilingStatus: Single, SSBenefits: 0, OtherIncome: 50000}, 0, "none"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeFederal(c.in, fr)
			if !approx(got.TaxableAmount, c.wantTaxable) {
				t.Errorf("taxable = %.2f, want %.2f", got.TaxableAmount, c.wantTaxable)
			}
			if got.Tier != c.wantTier {
				t.Errorf("tier = %q, want %q", got.Tier, c.wantTier)
			}
			if got.TaxableAmount > 0.85*c.in.SSBenefits+0.01 {
				t.Errorf("taxable %.2f exceeds 85%% cap", got.TaxableAmount)
			}
		})
	}
}

func TestMunicipalInterestOnlyRaisesTaxable(t *testing.T) {
	fr := rules2026().Federal
	base := Input{FilingStatus: Single, SSBenefits: 20000, OtherIncome: 20000}
	withMuni := base
	withMuni.TaxExemptInterest = 8000
	a := computeFederal(base, fr)
	b := computeFederal(withMuni, fr)
	if b.TaxableAmount <= a.TaxableAmount {
		t.Errorf("tax-exempt interest should raise taxable SS: %.2f -> %.2f", a.TaxableAmount, b.TaxableAmount)
	}
}
