// Package engine is the pure tax-rules engine: Social Security taxability at the
// federal level (IRS Publication 915 worksheet) and for the 8 states that tax
// Social Security benefits. It has NO HTTP or DB dependencies — inputs in,
// results out — so it can be exhaustively unit-tested.
//
// All thresholds and rates live in versioned rulesets (see rules.go) keyed by
// tax year, so the yearly update customers pay for is a data change, not a
// code rewrite.
package engine

import "fmt"

// FilingStatus enumerates the IRS filing statuses relevant to SS taxability.
// MFS is split because living together vs. apart yields very different rules.
type FilingStatus string

const (
	Single      FilingStatus = "single"
	HoH         FilingStatus = "hoh"
	MFJ         FilingStatus = "mfj"
	MFSApart    FilingStatus = "mfs_apart"    // married filing separately, lived apart all year
	MFSTogether FilingStatus = "mfs_together" // married filing separately, lived together
)

// ValidFilingStatuses is the canonical set, handy for input validation.
var ValidFilingStatuses = []FilingStatus{Single, HoH, MFJ, MFSApart, MFSTogether}

func (f FilingStatus) valid() bool {
	for _, v := range ValidFilingStatuses {
		if v == f {
			return true
		}
	}
	return false
}

// isJoint reports whether the status uses the married-filing-jointly thresholds.
func (f FilingStatus) isJoint() bool { return f == MFJ }

// Input is a single taxability computation request. Money values are annual USD.
type Input struct {
	FilingStatus      FilingStatus `json:"filing_status"`
	State             string       `json:"state"` // two-letter USPS code, e.g. "CO"; "" or non-taxing => no state tax
	Age               int          `json:"age"`
	AtFRA             bool         `json:"at_fra"` // at/above full retirement age (matters for RI)
	SSBenefits        float64      `json:"ss_benefits"`
	OtherIncome       float64      `json:"other_income"`        // AGI excluding SS
	TaxExemptInterest float64      `json:"tax_exempt_interest"` // municipal-bond interest
	TaxYear           int          `json:"tax_year"`
}

// Validate returns an error describing the first invalid field, or nil.
func (in Input) Validate() error {
	if !in.FilingStatus.valid() {
		return fmt.Errorf("invalid filing_status %q", in.FilingStatus)
	}
	if in.TaxYear < 1994 || in.TaxYear > 2100 {
		return fmt.Errorf("invalid tax_year %d", in.TaxYear)
	}
	if in.SSBenefits < 0 || in.OtherIncome < 0 || in.TaxExemptInterest < 0 {
		return fmt.Errorf("money inputs must be non-negative")
	}
	if in.Age < 0 || in.Age > 130 {
		return fmt.Errorf("invalid age %d", in.Age)
	}
	return nil
}

// agi is the taxpayer's approximate federal AGI used by state means tests:
// other income plus the federally taxable portion of benefits. States define
// AGI differently, but this is the figure their SS thresholds are written
// against.
func (in Input) agi(federallyTaxable float64) float64 {
	return in.OtherIncome + federallyTaxable
}

// FederalResult is the outcome of the federal computation.
type FederalResult struct {
	ProvisionalIncome float64 `json:"provisional_income"`
	TaxableAmount     float64 `json:"taxable_amount"` // taxable portion of SS benefits
	TaxablePercent    float64 `json:"taxable_percent"`
	Tier              string  `json:"tier"` // "none" | "up_to_50" | "up_to_85"
	Explanation       string  `json:"explanation"`
}

// StateResult is the outcome of a single state's computation.
type StateResult struct {
	State         string  `json:"state"`
	Taxable       bool    `json:"taxable"` // whether this state taxes SS at all
	TaxableAmount float64 `json:"taxable_amount"`
	Rate          float64 `json:"rate"` // marginal/flat rate applied (0..1)
	EstimatedTax  float64 `json:"estimated_tax"`
	Explanation   string  `json:"explanation"`
}

// Result is the full engine output for one Input.
type Result struct {
	TaxYear      int           `json:"tax_year"`
	Federal      FederalResult `json:"federal"`
	State        StateResult   `json:"state"`
	CombinedTax  float64       `json:"combined_tax"` // federal-taxable-driven state tax only (federal income tax itself is out of scope)
	Explanations []string      `json:"explanations"`
}

// round2 rounds to cents to keep money values clean across JSON boundaries.
func round2(v float64) float64 {
	if v < 0 {
		return -round2(-v)
	}
	return float64(int64(v*100+0.5)) / 100
}
