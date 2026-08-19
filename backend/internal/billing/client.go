// Package billing integrates Dodo Payments (a Merchant of Record) for hosted
// checkout and subscription webhooks. Only a stdlib HTTP client is used so the
// webhook signature verification — the security-critical path — is fully
// transparent and dependency-free.
//
// Dodo is a MoR: it collects and remits sales tax / VAT, so this app never
// touches tax-on-the-subscription itself.
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the Dodo Payments REST API.
type Client struct {
	apiKey        string
	webhookSecret string
	proProductID  string
	baseURL       string
	http          *http.Client
}

// New builds a Dodo client. env is "test_mode" or "live_mode" and selects the
// API host. When apiKey is empty the client is considered unconfigured and
// checkout calls return ErrNotConfigured.
func New(apiKey, webhookSecret, proProductID, env string) *Client {
	base := "https://test.dodopayments.com"
	if env == "live_mode" {
		base = "https://live.dodopayments.com"
	}
	return &Client{
		apiKey:        apiKey,
		webhookSecret: webhookSecret,
		proProductID:  proProductID,
		baseURL:       base,
		http:          &http.Client{Timeout: 20 * time.Second},
	}
}

// ErrNotConfigured is returned when billing env vars are missing.
var ErrNotConfigured = errors.New("billing is not configured (set DODO_PAYMENTS_API_KEY and DODO_PRO_PRODUCT_ID)")

// Configured reports whether checkout can be created.
func (c *Client) Configured() bool { return c.apiKey != "" && c.proProductID != "" }

// WebhookSecret exposes the configured signing secret (may be empty).
func (c *Client) WebhookSecret() string { return c.webhookSecret }

// CheckoutParams describes a Pro checkout for one user.
type CheckoutParams struct {
	UserID    string
	Email     string
	FirmName  string
	ReturnURL string
}

// CreateCheckoutSession creates a hosted Dodo checkout for the Pro product and
// returns the URL the browser should be redirected to. The user id and email
// are attached as metadata so the resulting webhook can be tied back to the
// account.
func (c *Client) CreateCheckoutSession(ctx context.Context, p CheckoutParams) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	body := map[string]any{
		"product_cart": []map[string]any{
			{"product_id": c.proProductID, "quantity": 1},
		},
		"customer": map[string]any{
			"email": p.Email,
			"name":  p.FirmName,
		},
		"return_url": p.ReturnURL,
		"metadata": map[string]string{
			"user_id": p.UserID,
		},
	}
	raw, err := c.post(ctx, "/checkouts", body)
	if err != nil {
		return "", err
	}
	// Dodo returns the hosted URL under one of these keys depending on the
	// endpoint/version; accept whichever is present.
	var resp struct {
		CheckoutURL string `json:"checkout_url"`
		PaymentLink string `json:"payment_link"`
		Link        string `json:"link"`
		URL         string `json:"url"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("decode checkout response: %w", err)
	}
	for _, u := range []string{resp.CheckoutURL, resp.PaymentLink, resp.Link, resp.URL} {
		if u != "" {
			return u, nil
		}
	}
	return "", fmt.Errorf("checkout response contained no redirect URL: %s", string(raw))
}

// SubscriptionInfo is the subset of a Dodo subscription this app tracks.
type SubscriptionInfo struct {
	Status        string     `json:"status"`
	CustomerID    string     `json:"-"`
	NextBillingAt string     `json:"next_billing_date"`
	PeriodEnd     *time.Time `json:"-"`
}

// GetSubscription fetches current subscription state from Dodo. Used by the
// manual reconcile endpoint when webhooks can't reach localhost.
func (c *Client) GetSubscription(ctx context.Context, id string) (*SubscriptionInfo, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	raw, err := c.get(ctx, "/subscriptions/"+id)
	if err != nil {
		return nil, err
	}
	var body struct {
		Status        string `json:"status"`
		NextBillingAt string `json:"next_billing_date"`
		Customer      struct {
			CustomerID string `json:"customer_id"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	info := &SubscriptionInfo{Status: body.Status, CustomerID: body.Customer.CustomerID}
	if body.NextBillingAt != "" {
		for _, l := range []string{time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(l, body.NextBillingAt); err == nil {
				info.PeriodEnd = &t
				break
			}
		}
	}
	return info, nil
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("dodo GET %s -> %d: %s", path, res.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("dodo %s -> %d: %s", path, res.StatusCode, string(data))
	}
	return data, nil
}
