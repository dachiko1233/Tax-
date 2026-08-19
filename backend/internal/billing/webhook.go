package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrBadSignature means the webhook could not be authenticated and must be
// rejected before any state is trusted.
var ErrBadSignature = errors.New("invalid webhook signature")

// tolerance bounds how far the webhook timestamp may drift from now.
const tolerance = 5 * time.Minute

// VerifySignature authenticates a Dodo webhook using the Standard Webhooks
// scheme (the same HMAC-SHA256 construction Dodo/Svix use):
//
//	signed = "{webhook-id}.{webhook-timestamp}.{body}"
//	sig    = base64(HMAC_SHA256(secretBytes, signed))
//
// The secret is the part after the optional "whsec_" prefix, base64-decoded.
// The webhook-signature header may carry several space-separated
// "v1,<sig>" entries; a match against any one passes.
func (c *Client) VerifySignature(id, timestamp, header string, body []byte) error {
	if c.webhookSecret == "" {
		return errors.New("webhook secret not configured")
	}
	if id == "" || timestamp == "" || header == "" {
		return ErrBadSignature
	}

	// Reject stale/future timestamps to blunt replay attacks.
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrBadSignature
	}
	drift := time.Since(time.Unix(ts, 0))
	if drift > tolerance || drift < -tolerance {
		return fmt.Errorf("%w: timestamp outside tolerance", ErrBadSignature)
	}

	secret := strings.TrimPrefix(c.webhookSecret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		// Some deployments use a raw (non-base64) secret; fall back to bytes.
		key = []byte(secret)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	for _, part := range strings.Split(header, " ") {
		got := part
		if i := strings.IndexByte(part, ','); i >= 0 {
			got = part[i+1:] // strip the "v1," version prefix
		}
		if hmac.Equal([]byte(got), []byte(expected)) {
			return nil
		}
	}
	return ErrBadSignature
}

// Event is the decoded webhook envelope. Only the fields the app acts on are
// modeled; the rest of the payload is ignored.
type Event struct {
	Type string `json:"type"`
	Data struct {
		SubscriptionID string `json:"subscription_id"`
		Status         string `json:"status"`
		NextBillingAt  string `json:"next_billing_date"`
		Customer       struct {
			CustomerID string `json:"customer_id"`
			Email      string `json:"email"`
		} `json:"customer"`
		Metadata map[string]string `json:"metadata"`
	} `json:"data"`
}

// ParseEvent decodes a verified webhook body.
func ParseEvent(body []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// PeriodEnd parses the next-billing timestamp if present.
func (e *Event) PeriodEnd() *time.Time {
	if e.Data.NextBillingAt == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.999999Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, e.Data.NextBillingAt); err == nil {
			return &t
		}
	}
	return nil
}
