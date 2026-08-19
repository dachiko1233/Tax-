// Package models holds the database-backed record structs shared across the API
// and db layers.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User is a paying accountant account.
type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"` // never serialized
	FirmName       string    `json:"firm_name"`
	EmailVerified  bool      `json:"email_verified"`
	Plan           string    `json:"plan"` // "free" | "pro"
	DodoCustomerID string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

// Subscription mirrors a Dodo subscription; webhooks keep it current and it is
// the source of truth for Pro entitlement.
type Subscription struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	DodoSubscriptionID string     `json:"dodo_subscription_id"`
	Status             string     `json:"status"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Client is one of an accountant's retiree clients.
type Client struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	Name              string    `json:"name"`
	FilingStatus      string    `json:"filing_status"`
	State             string    `json:"state"`
	Age               int       `json:"age"`
	AtFRA             bool      `json:"at_fra"`
	SSBenefits        float64   `json:"ss_benefits"`
	OtherIncome       float64   `json:"other_income"`
	TaxExemptInterest float64   `json:"tax_exempt_interest"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Scenario is a saved what-if calculation for a client.
type Scenario struct {
	ID          uuid.UUID      `json:"id"`
	ClientID    uuid.UUID      `json:"client_id"`
	TaxYear     int            `json:"tax_year"`
	Label       string         `json:"label"`
	InputsJSON  map[string]any `json:"inputs"`
	ResultsJSON map[string]any `json:"results"`
	CreatedAt   time.Time      `json:"created_at"`
}
