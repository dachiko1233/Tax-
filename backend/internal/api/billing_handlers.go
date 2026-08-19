package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ss-tax-engine/internal/billing"
	"ss-tax-engine/internal/models"
)

// handleCheckout creates a Dodo hosted checkout session for the Pro plan and
// returns the URL the frontend should redirect the browser to.
func (s *Server) handleCheckout(c *gin.Context) {
	uid := userID(c)
	u, err := s.DB.GetUserByID(context.Background(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if u.Plan == PlanPro {
		c.JSON(http.StatusConflict, gin.H{"error": "already on the Pro plan"})
		return
	}
	if !s.Bill.Configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "billing is not configured on this server; set the Dodo env vars",
		})
		return
	}
	url, err := s.Bill.CreateCheckoutSession(context.Background(), billing.CheckoutParams{
		UserID:    u.ID.String(),
		Email:     u.Email,
		FirmName:  u.FirmName,
		ReturnURL: s.AppBaseURL + "/billing?status=success",
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not create checkout session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"checkout_url": url})
}

// handleBillingStatus returns the user's plan and subscription state.
func (s *Server) handleBillingStatus(c *gin.Context) {
	u, err := s.DB.GetUserByID(context.Background(), userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	out := gin.H{
		"plan":               u.Plan,
		"billing_configured": s.Bill.Configured(),
		"subscription":       nil,
	}
	if sub, err := s.DB.GetSubscriptionByUser(context.Background(), u.ID); err == nil {
		out["subscription"] = sub
	}
	c.JSON(http.StatusOK, out)
}

// handleBillingVerify is a manual reconcile for local development, where Dodo
// webhooks usually can't reach localhost. In configured mode it re-fetches the
// user's subscription from Dodo and syncs entitlement. With billing NOT
// configured it grants Pro locally so the Pro experience can be exercised
// without real payment.
func (s *Server) handleBillingVerify(c *gin.Context) {
	uid := userID(c)
	u, err := s.DB.GetUserByID(context.Background(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	if !s.Bill.Configured() {
		if err := s.DB.SetUserPlan(context.Background(), uid, PlanPro); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"plan":     PlanPro,
			"dev_mode": true,
			"message":  "Billing not configured; granted Pro for local development.",
		})
		return
	}

	sub, err := s.DB.GetSubscriptionByUser(context.Background(), uid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"plan": u.Plan, "message": "no subscription on file yet"})
		return
	}
	info, err := s.Bill.GetSubscription(context.Background(), sub.DodoSubscriptionID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not reach billing provider"})
		return
	}
	plan := planForStatus(info.Status)
	_ = s.DB.UpsertSubscription(context.Background(), &models.Subscription{
		UserID:             uid,
		DodoSubscriptionID: sub.DodoSubscriptionID,
		Status:             info.Status,
		CurrentPeriodEnd:   info.PeriodEnd,
	})
	_ = s.DB.SetUserPlan(context.Background(), uid, plan)
	c.JSON(http.StatusOK, gin.H{"plan": plan, "status": info.Status})
}

// handleDodoWebhook is the source of truth for entitlement. It verifies the
// Standard-Webhooks signature BEFORE trusting any payload, then updates the
// user's plan and subscription record.
func (s *Server) handleDodoWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read body"})
		return
	}
	if s.Bill.WebhookSecret() == "" {
		// Refuse to process unauthenticated webhooks.
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook secret not configured"})
		return
	}
	if err := s.Bill.VerifySignature(
		c.GetHeader("webhook-id"),
		c.GetHeader("webhook-timestamp"),
		c.GetHeader("webhook-signature"),
		body,
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	evt, err := billing.ParseEvent(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	uid, ok := s.resolveWebhookUser(evt)
	if !ok {
		// Acknowledge so Dodo stops retrying, but there's nothing to update.
		c.JSON(http.StatusOK, gin.H{"status": "ignored: unknown user"})
		return
	}

	ctx := context.Background()
	if evt.Data.Customer.CustomerID != "" {
		_ = s.DB.SetDodoCustomerID(ctx, uid, evt.Data.Customer.CustomerID)
	}
	if evt.Data.SubscriptionID != "" {
		_ = s.DB.UpsertSubscription(ctx, &models.Subscription{
			UserID:             uid,
			DodoSubscriptionID: evt.Data.SubscriptionID,
			Status:             webhookStatus(evt),
			CurrentPeriodEnd:   evt.PeriodEnd(),
		})
	}

	// Entitlement transitions. subscription.updated is only acted on when it
	// carries an explicit status, so an ambiguous update never silently
	// downgrades a paying user.
	switch evt.Type {
	case "subscription.active", "subscription.renewed":
		_ = s.DB.SetUserPlan(ctx, uid, PlanPro)
	case "subscription.on_hold", "subscription.failed", "subscription.cancelled", "subscription.expired":
		_ = s.DB.SetUserPlan(ctx, uid, PlanFree)
	case "subscription.updated":
		if evt.Data.Status != "" {
			_ = s.DB.SetUserPlan(ctx, uid, planForStatus(evt.Data.Status))
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// resolveWebhookUser ties an event back to a local user via the metadata we set
// at checkout, falling back to the customer email.
func (s *Server) resolveWebhookUser(evt *billing.Event) (uuid.UUID, bool) {
	if raw := evt.Data.Metadata["user_id"]; raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return id, true
		}
	}
	if email := evt.Data.Customer.Email; email != "" {
		if u, err := s.DB.GetUserByEmail(context.Background(), email); err == nil {
			return u.ID, true
		}
	}
	return uuid.Nil, false
}

// planForStatus maps a subscription status string to an entitlement plan.
func planForStatus(status string) string {
	switch status {
	case "active", "renewed":
		return PlanPro
	default:
		return PlanFree
	}
}

// webhookStatus derives a stored status from the event, preferring an explicit
// data.status and otherwise deriving it from the event type.
func webhookStatus(evt *billing.Event) string {
	if evt.Data.Status != "" {
		return evt.Data.Status
	}
	switch evt.Type {
	case "subscription.active", "subscription.renewed":
		return "active"
	case "subscription.on_hold":
		return "on_hold"
	case "subscription.failed":
		return "failed"
	default:
		return "updated"
	}
}
