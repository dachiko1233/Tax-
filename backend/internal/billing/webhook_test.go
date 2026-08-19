package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// sign produces a valid Standard-Webhooks signature header for the given parts.
func sign(secretB64, id, ts string, body []byte) string {
	key, _ := base64.StdEncoding.DecodeString(secretB64)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	rawSecret := base64.StdEncoding.EncodeToString([]byte("super-secret-key-material"))
	c := New("k", "whsec_"+rawSecret, "prod_x", "test_mode")

	id := "msg_123"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := []byte(`{"type":"subscription.active","data":{"subscription_id":"sub_1"}}`)
	good := sign(rawSecret, id, ts, body)

	t.Run("valid", func(t *testing.T) {
		if err := c.VerifySignature(id, ts, good, body); err != nil {
			t.Fatalf("expected valid signature, got %v", err)
		}
	})

	t.Run("tampered body", func(t *testing.T) {
		if err := c.VerifySignature(id, ts, good, []byte(`{"type":"hacked"}`)); err == nil {
			t.Fatal("expected failure on tampered body")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		other := New("k", "whsec_"+base64.StdEncoding.EncodeToString([]byte("different")), "prod_x", "test_mode")
		if err := other.VerifySignature(id, ts, good, body); err == nil {
			t.Fatal("expected failure with wrong secret")
		}
	})

	t.Run("stale timestamp", func(t *testing.T) {
		oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		staleSig := sign(rawSecret, id, oldTs, body)
		if err := c.VerifySignature(id, oldTs, staleSig, body); err == nil {
			t.Fatal("expected failure on stale timestamp")
		}
	})

	t.Run("multiple signatures, one valid", func(t *testing.T) {
		hdr := fmt.Sprintf("v1,%s %s", "aGVsbG8=", good[len("v1,"):])
		if err := c.VerifySignature(id, ts, hdr, body); err != nil {
			t.Fatalf("expected match among multiple sigs, got %v", err)
		}
	})
}

func TestParseEvent(t *testing.T) {
	body := []byte(`{"type":"subscription.active","data":{"subscription_id":"sub_9","status":"active","customer":{"customer_id":"cus_1","email":"a@b.com"},"metadata":{"user_id":"u-1"},"next_billing_date":"2027-01-01"}}`)
	e, err := ParseEvent(body)
	if err != nil {
		t.Fatal(err)
	}
	if e.Type != "subscription.active" || e.Data.SubscriptionID != "sub_9" {
		t.Fatalf("bad parse: %+v", e)
	}
	if e.Data.Metadata["user_id"] != "u-1" || e.Data.Customer.Email != "a@b.com" {
		t.Fatalf("bad metadata/customer: %+v", e.Data)
	}
	if pe := e.PeriodEnd(); pe == nil || pe.Year() != 2027 {
		t.Fatalf("bad period end: %v", pe)
	}
}
