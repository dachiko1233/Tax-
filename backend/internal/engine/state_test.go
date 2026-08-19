package engine

import "testing"

// helper: run computeState against the 2026 ruleset with an explicit
// federally-taxable amount, letting the test set AGI via OtherIncome.
func stateCase(t *testing.T, in Input, fedTaxable float64) StateResult {
	t.Helper()
	rs := rules2026()
	return computeState(in, rs, fedTaxable)
}

func TestNonTaxingState(t *testing.T) {
	r := stateCase(t, Input{FilingStatus: Single, State: "FL"}, 17000)
	if r.Taxable || r.EstimatedTax != 0 {
		t.Errorf("FL should not tax SS: %+v", r)
	}
	empty := stateCase(t, Input{FilingStatus: Single, State: ""}, 17000)
	if empty.Taxable {
		t.Errorf("empty state should not be taxable")
	}
}

func TestColorado(t *testing.T) {
	over := stateCase(t, Input{FilingStatus: Single, State: "CO", Age: 70}, 17000)
	if over.TaxableAmount != 0 || over.EstimatedTax != 0 {
		t.Errorf("CO 65+ should fully subtract, got %+v", over)
	}
	under := stateCase(t, Input{FilingStatus: Single, State: "CO", Age: 60}, 17000)
	if !approx(under.TaxableAmount, 17000) || !approx(under.EstimatedTax, 748) {
		t.Errorf("CO under-65 tax = %.2f (taxable %.2f), want 748", under.EstimatedTax, under.TaxableAmount)
	}
}

func TestMontana(t *testing.T) {
	old := stateCase(t, Input{FilingStatus: Single, State: "MT", Age: 70}, 10000)
	if !approx(old.TaxableAmount, 4500) || !approx(old.EstimatedTax, 265.5) {
		t.Errorf("MT 65+ = taxable %.2f tax %.2f, want 4500 / 265.5", old.TaxableAmount, old.EstimatedTax)
	}
	young := stateCase(t, Input{FilingStatus: Single, State: "MT", Age: 60}, 10000)
	if !approx(young.TaxableAmount, 10000) || !approx(young.EstimatedTax, 590) {
		t.Errorf("MT under-65 = taxable %.2f tax %.2f, want 10000 / 590", young.TaxableAmount, young.EstimatedTax)
	}
}

func TestConnecticutBands(t *testing.T) {
	// cap 75000 single, width 25000, rate 5%. fedTaxable = 10000.
	// below cap: AGI = 60000 + 10000 = 70000
	below := stateCase(t, Input{FilingStatus: Single, State: "CT", OtherIncome: 60000}, 10000)
	if below.TaxableAmount != 0 {
		t.Errorf("CT below cap should be exempt, got %+v", below)
	}
	// midpoint: AGI = 77500 + 10000 = 87500 => 50% => taxable 5000, tax 250
	mid := stateCase(t, Input{FilingStatus: Single, State: "CT", OtherIncome: 77500}, 10000)
	if !approx(mid.TaxableAmount, 5000) || !approx(mid.EstimatedTax, 250) {
		t.Errorf("CT midpoint = taxable %.2f tax %.2f, want 5000 / 250", mid.TaxableAmount, mid.EstimatedTax)
	}
	// above band: AGI = 100000 + 10000 = 110000 => full => taxable 10000, tax 500
	above := stateCase(t, Input{FilingStatus: Single, State: "CT", OtherIncome: 100000}, 10000)
	if !approx(above.TaxableAmount, 10000) || !approx(above.EstimatedTax, 500) {
		t.Errorf("CT above band = taxable %.2f tax %.2f, want 10000 / 500", above.TaxableAmount, above.EstimatedTax)
	}
}

func TestNewMexicoCliff(t *testing.T) {
	// cap 100000 single, width 0 => hard cliff. fedTaxable 12000.
	below := stateCase(t, Input{FilingStatus: Single, State: "NM", OtherIncome: 80000}, 12000) // AGI 92000
	if below.TaxableAmount != 0 {
		t.Errorf("NM below cap should be exempt, got %+v", below)
	}
	above := stateCase(t, Input{FilingStatus: Single, State: "NM", OtherIncome: 95000}, 12000) // AGI 107000
	if !approx(above.TaxableAmount, 12000) {
		t.Errorf("NM above cap should be fully taxable, got %+v", above)
	}
}

func TestRhodeIslandFRA(t *testing.T) {
	// exempt requires FRA AND AGI below cap 104200 single.
	notFRA := stateCase(t, Input{FilingStatus: Single, State: "RI", AtFRA: false, OtherIncome: 30000}, 12000)
	if notFRA.TaxableAmount == 0 {
		t.Errorf("RI without FRA should be taxable even at low AGI, got %+v", notFRA)
	}
	atFRALow := stateCase(t, Input{FilingStatus: Single, State: "RI", AtFRA: true, OtherIncome: 30000}, 12000) // AGI 42000
	if atFRALow.TaxableAmount != 0 {
		t.Errorf("RI at FRA below cap should be exempt, got %+v", atFRALow)
	}
	atFRAHigh := stateCase(t, Input{FilingStatus: Single, State: "RI", AtFRA: true, OtherIncome: 120000}, 12000) // AGI 132000 > cap
	if atFRAHigh.TaxableAmount == 0 {
		t.Errorf("RI at FRA above cap should be taxable, got %+v", atFRAHigh)
	}
}

func TestMFJUsesJointCaps(t *testing.T) {
	// Vermont: single cap 65000, mfj cap 80000. AGI 75000 => exempt for MFJ, taxable for single.
	single := stateCase(t, Input{FilingStatus: Single, State: "VT", OtherIncome: 65000}, 10000) // AGI 75000 > 65000
	if single.TaxableAmount == 0 {
		t.Errorf("VT single at AGI 75000 should be taxable, got %+v", single)
	}
	joint := stateCase(t, Input{FilingStatus: MFJ, State: "VT", OtherIncome: 65000}, 10000) // AGI 75000 < 80000
	if joint.TaxableAmount != 0 {
		t.Errorf("VT MFJ at AGI 75000 should be exempt, got %+v", joint)
	}
}

func TestEngineEndToEnd(t *testing.T) {
	e := New(DefaultProvider())
	res, err := e.Calculate(Input{
		FilingStatus: Single, State: "CO", Age: 60,
		SSBenefits: 20000, OtherIncome: 40000, TaxYear: 2026,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Federal.Tier != "up_to_85" {
		t.Errorf("federal tier = %s", res.Federal.Tier)
	}
	if len(res.Explanations) != 2 {
		t.Errorf("want federal + state explanation, got %d", len(res.Explanations))
	}
	if !approx(res.CombinedTax, res.State.EstimatedTax) {
		t.Errorf("combined tax mismatch")
	}
}

func TestUnknownYear(t *testing.T) {
	e := New(DefaultProvider())
	if _, err := e.Calculate(Input{FilingStatus: Single, TaxYear: 1999}); err == nil {
		t.Errorf("expected error for unseeded year")
	}
}
