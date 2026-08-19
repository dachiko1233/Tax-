package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"ss-tax-engine/internal/models"
)

// --- users ---

const userCols = `id, email, password_hash, firm_name, email_verified, plan, dodo_customer_id, created_at`

func scanUser(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirmName,
		&u.EmailVerified, &u.Plan, &u.DodoCustomerID, &u.CreatedAt)
	return u, err
}

func (d *DB) CreateUser(ctx context.Context, email, passwordHash, firmName string) (*models.User, error) {
	return scanUser(d.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, firm_name)
		VALUES ($1, $2, $3)
		RETURNING `+userCols, email, passwordHash, firmName))
}

func (d *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u, err := scanUser(d.Pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, email))
	return u, mapNoRows(err)
}

func (d *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, err := scanUser(d.Pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1`, id))
	return u, mapNoRows(err)
}

// SetEmailVerified marks a user's email as verified.
func (d *DB) SetEmailVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE users SET email_verified = true WHERE id = $1`, userID)
	return err
}

// SetUserPlan updates the entitlement plan ("free"|"pro").
func (d *DB) SetUserPlan(ctx context.Context, userID uuid.UUID, plan string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE users SET plan = $2 WHERE id = $1`, userID, plan)
	return err
}

// SetDodoCustomerID stores the Dodo customer id the first time we learn it.
func (d *DB) SetDodoCustomerID(ctx context.Context, userID uuid.UUID, customerID string) error {
	_, err := d.Pool.Exec(ctx,
		`UPDATE users SET dodo_customer_id = $2 WHERE id = $1`, userID, customerID)
	return err
}

// --- email verification tokens ---

// CreateVerificationToken stores the SHA-256 hash of a token with an expiry.
func (d *DB) CreateVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, userID, tokenHash, expiresAt)
	return err
}

// ConsumeVerificationToken atomically validates and consumes a token, returning
// the owning user id. It fails if the token is unknown, expired, or already
// consumed.
func (d *DB) ConsumeVerificationToken(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := d.Pool.QueryRow(ctx, `
		UPDATE email_verification_tokens
		SET consumed_at = now()
		WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		RETURNING user_id`, tokenHash).Scan(&userID)
	return userID, mapNoRows(err)
}

// RecentTokenCount counts verification tokens issued to a user since `since`,
// used to rate-limit the "resend verification" endpoint.
func (d *DB) RecentTokenCount(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx, `
		SELECT count(*) FROM email_verification_tokens
		WHERE user_id = $1 AND created_at > $2`, userID, since).Scan(&n)
	return n, err
}

// --- plan-limit counts ---

// CountClients returns how many clients the user currently has.
func (d *DB) CountClients(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM clients WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

// CountScenarios returns how many scenarios exist for a client.
func (d *DB) CountScenarios(ctx context.Context, clientID uuid.UUID) (int, error) {
	var n int
	err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM scenarios WHERE client_id = $1`, clientID).Scan(&n)
	return n, err
}

// --- subscriptions ---

// UpsertSubscription inserts or updates a subscription keyed by its Dodo id.
func (d *DB) UpsertSubscription(ctx context.Context, s *models.Subscription) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, dodo_subscription_id, status, current_period_end)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (dodo_subscription_id)
		DO UPDATE SET status = EXCLUDED.status,
		              current_period_end = EXCLUDED.current_period_end,
		              updated_at = now()`,
		s.UserID, s.DodoSubscriptionID, s.Status, s.CurrentPeriodEnd)
	return err
}

// GetSubscriptionByUser returns the most recent subscription for a user, or
// ErrNotFound if there is none.
func (d *DB) GetSubscriptionByUser(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	s := &models.Subscription{}
	err := d.Pool.QueryRow(ctx, `
		SELECT id, user_id, dodo_subscription_id, status, current_period_end, created_at, updated_at
		FROM subscriptions WHERE user_id = $1
		ORDER BY updated_at DESC LIMIT 1`, userID,
	).Scan(&s.ID, &s.UserID, &s.DodoSubscriptionID, &s.Status,
		&s.CurrentPeriodEnd, &s.CreatedAt, &s.UpdatedAt)
	return s, mapNoRows(err)
}

// --- clients ---

func (d *DB) ListClients(ctx context.Context, userID uuid.UUID) ([]models.Client, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, user_id, name, filing_status, state, age, at_fra,
		       ss_benefits, other_income, tax_exempt_interest, created_at, updated_at
		FROM clients WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Client
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.FilingStatus, &c.State,
			&c.Age, &c.AtFRA, &c.SSBenefits, &c.OtherIncome, &c.TaxExemptInterest,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) GetClient(ctx context.Context, userID, id uuid.UUID) (*models.Client, error) {
	c := &models.Client{}
	err := d.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, filing_status, state, age, at_fra,
		       ss_benefits, other_income, tax_exempt_interest, created_at, updated_at
		FROM clients WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.FilingStatus, &c.State, &c.Age, &c.AtFRA,
		&c.SSBenefits, &c.OtherIncome, &c.TaxExemptInterest, &c.CreatedAt, &c.UpdatedAt)
	return c, mapNoRows(err)
}

func (d *DB) CreateClient(ctx context.Context, c *models.Client) (*models.Client, error) {
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO clients (user_id, name, filing_status, state, age, at_fra,
		                     ss_benefits, other_income, tax_exempt_interest)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at`,
		c.UserID, c.Name, c.FilingStatus, c.State, c.Age, c.AtFRA,
		c.SSBenefits, c.OtherIncome, c.TaxExemptInterest,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (d *DB) UpdateClient(ctx context.Context, userID uuid.UUID, c *models.Client) (*models.Client, error) {
	err := d.Pool.QueryRow(ctx, `
		UPDATE clients SET name=$1, filing_status=$2, state=$3, age=$4, at_fra=$5,
		       ss_benefits=$6, other_income=$7, tax_exempt_interest=$8, updated_at=now()
		WHERE id=$9 AND user_id=$10
		RETURNING id, user_id, created_at, updated_at`,
		c.Name, c.FilingStatus, c.State, c.Age, c.AtFRA,
		c.SSBenefits, c.OtherIncome, c.TaxExemptInterest, c.ID, userID,
	).Scan(&c.ID, &c.UserID, &c.CreatedAt, &c.UpdatedAt)
	return c, mapNoRows(err)
}

func (d *DB) DeleteClient(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := d.Pool.Exec(ctx, `DELETE FROM clients WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- scenarios ---

// ClientOwnedBy verifies the client belongs to the user before scenario ops.
func (d *DB) ClientOwnedBy(ctx context.Context, userID, clientID uuid.UUID) (bool, error) {
	var exists bool
	err := d.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM clients WHERE id=$1 AND user_id=$2)`,
		clientID, userID).Scan(&exists)
	return exists, err
}

func (d *DB) CreateScenario(ctx context.Context, s *models.Scenario) (*models.Scenario, error) {
	inputs, _ := json.Marshal(s.InputsJSON)
	results, _ := json.Marshal(s.ResultsJSON)
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO scenarios (client_id, tax_year, label, inputs_json, results_json)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		s.ClientID, s.TaxYear, s.Label, inputs, results,
	).Scan(&s.ID, &s.CreatedAt)
	return s, err
}

func (d *DB) ListScenarios(ctx context.Context, clientID uuid.UUID) ([]models.Scenario, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, client_id, tax_year, label, inputs_json, results_json, created_at
		FROM scenarios WHERE client_id = $1 ORDER BY created_at DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Scenario
	for rows.Next() {
		var s models.Scenario
		var inputs, results []byte
		if err := rows.Scan(&s.ID, &s.ClientID, &s.TaxYear, &s.Label,
			&inputs, &results, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(inputs, &s.InputsJSON)
		_ = json.Unmarshal(results, &s.ResultsJSON)
		out = append(out, s)
	}
	return out, rows.Err()
}

// --- tax rules (seed + load) ---

// UpsertTaxRule stores a versioned ruleset row.
func (d *DB) UpsertTaxRule(ctx context.Context, taxYear int, jurisdiction string, rulesJSON []byte) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO tax_rules (tax_year, jurisdiction, rules_json, effective_from)
		VALUES ($1,$2,$3, make_date($1,1,1))
		ON CONFLICT (tax_year, jurisdiction)
		DO UPDATE SET rules_json = EXCLUDED.rules_json`,
		taxYear, jurisdiction, rulesJSON)
	return err
}
