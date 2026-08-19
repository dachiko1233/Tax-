package engine

import "fmt"

// FederalRules holds the provisional-income thresholds. These have been fixed in
// statute since 1994 and are NOT inflation-adjusted, but they are versioned here
// anyway so a future law change is a data edit.
type FederalRules struct {
	// Base and Upper are the two provisional-income thresholds per filing status.
	Base  map[FilingStatus]float64 `json:"base"`
	Upper map[FilingStatus]float64 `json:"upper"`
}

// stateKind selects which state algorithm applies. Keeping the algorithm as a
// tagged kind (rather than one function per state hardcoding numbers) lets every
// state's thresholds/rates stay pure data in the ruleset.
type stateKind string

const (
	// kindExemptBelowCap: fully exempt at/below the AGI cap, then the taxable
	// fraction phases in linearly across PhaseOutWidth (Width 0 = hard cliff to
	// fully taxable). Base for tax is the federally taxable SS amount.
	kindExemptBelowCap stateKind = "exempt_below_cap"
	// kindColorado: age >= AgeThreshold => full subtraction (0 taxable);
	// otherwise the federally taxable amount is taxed at FlatRate.
	kindColorado stateKind = "colorado"
	// kindMontana: follows federal inclusion; age >= AgeThreshold gets a fixed
	// SubtractionAmount off the taxable base.
	kindMontana stateKind = "montana"
	// kindFRAExempt: like exempt_below_cap but the exemption ALSO requires the
	// taxpayer to be at full retirement age (Rhode Island).
	kindFRAExempt stateKind = "fra_exempt"
)

// StateRules is the parameter set for one taxing state in one year.
type StateRules struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Kind          stateKind `json:"kind"`
	FlatRate      float64   `json:"flat_rate"`       // representative income-tax rate applied to the taxable SS portion
	AGICapSingle  float64   `json:"agi_cap_single"`  // full-exemption AGI cap for single/HoH/MFS
	AGICapMFJ     float64   `json:"agi_cap_mfj"`     // full-exemption AGI cap for MFJ
	PhaseOutWidth float64   `json:"phase_out_width"` // width of the phase-in band above the cap; 0 => cliff
	Subtraction   float64   `json:"subtraction"`     // fixed subtraction amount (Montana)
	AgeThreshold  int       `json:"age_threshold"`   // age-based benefit cutoff (Colorado, Montana)
}

// cap returns the full-exemption AGI cap for the given filing status.
func (s StateRules) cap(fs FilingStatus) float64 {
	if fs.isJoint() {
		return s.AGICapMFJ
	}
	return s.AGICapSingle
}

// RuleSet is the complete set of rules for a single tax year.
type RuleSet struct {
	TaxYear int                    `json:"tax_year"`
	Federal FederalRules           `json:"federal"`
	States  map[string]*StateRules `json:"states"` // keyed by two-letter code
}

// Provider looks up the ruleset for a tax year. The DB-backed loader implements
// the same shape; the built-in default below is the seed source of truth.
type Provider interface {
	ForYear(year int) (*RuleSet, error)
}

// StaticProvider serves rulesets from an in-memory map (used for tests and as
// the seed for the tax_rules table).
type StaticProvider struct {
	Years map[int]*RuleSet
}

func (p *StaticProvider) ForYear(year int) (*RuleSet, error) {
	rs, ok := p.Years[year]
	if !ok {
		return nil, fmt.Errorf("no tax rules loaded for year %d", year)
	}
	return rs, nil
}

// DefaultProvider returns a provider seeded with the built-in rulesets.
func DefaultProvider() *StaticProvider {
	return &StaticProvider{Years: map[int]*RuleSet{
		2026: rules2026(),
	}}
}

// rules2026 is the built-in 2026 starting point.
//
// ⚠️  The federal thresholds are statutory and stable. The STATE thresholds and
// especially the representative FlatRate values are a 2026 STARTING POINT and
// MUST be confirmed against each state revenue department before shipping — they
// change yearly. They live here (and are seeded into tax_rules) precisely so
// that confirmation is a data edit.
func rules2026() *RuleSet {
	return &RuleSet{
		TaxYear: 2026,
		Federal: FederalRules{
			Base: map[FilingStatus]float64{
				Single: 25000, HoH: 25000, MFSApart: 25000,
				MFJ: 32000, MFSTogether: 0,
			},
			Upper: map[FilingStatus]float64{
				Single: 34000, HoH: 34000, MFSApart: 34000,
				MFJ: 44000, MFSTogether: 0,
			},
		},
		States: map[string]*StateRules{
			"CO": {Code: "CO", Name: "Colorado", Kind: kindColorado,
				FlatRate: 0.044, AgeThreshold: 65},
			"CT": {Code: "CT", Name: "Connecticut", Kind: kindExemptBelowCap,
				FlatRate: 0.05, AGICapSingle: 75000, AGICapMFJ: 100000, PhaseOutWidth: 25000},
			"MN": {Code: "MN", Name: "Minnesota", Kind: kindExemptBelowCap,
				FlatRate: 0.068, AGICapSingle: 86410, AGICapMFJ: 110780, PhaseOutWidth: 35000},
			"MT": {Code: "MT", Name: "Montana", Kind: kindMontana,
				FlatRate: 0.059, Subtraction: 5500, AgeThreshold: 65},
			"NM": {Code: "NM", Name: "New Mexico", Kind: kindExemptBelowCap,
				FlatRate: 0.049, AGICapSingle: 100000, AGICapMFJ: 150000, PhaseOutWidth: 0},
			"RI": {Code: "RI", Name: "Rhode Island", Kind: kindFRAExempt,
				FlatRate: 0.0475, AGICapSingle: 104200, AGICapMFJ: 130250, PhaseOutWidth: 0},
			"UT": {Code: "UT", Name: "Utah", Kind: kindExemptBelowCap,
				FlatRate: 0.0445, AGICapSingle: 54000, AGICapMFJ: 90000, PhaseOutWidth: 20000},
			"VT": {Code: "VT", Name: "Vermont", Kind: kindExemptBelowCap,
				FlatRate: 0.066, AGICapSingle: 65000, AGICapMFJ: 80000, PhaseOutWidth: 15000},
		},
	}
}

// TaxingStates lists the state codes that tax Social Security benefits in the
// built-in ruleset. Everything else (42 states + DC) returns a non-taxable
// StateResult.
func TaxingStates() []string {
	return []string{"CO", "CT", "MN", "MT", "NM", "RI", "UT", "VT"}
}
