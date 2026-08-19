package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// planFor returns the user's entitlement plan ("free"|"pro").
func (s *Server) planFor(ctx context.Context, uid uuid.UUID) (string, error) {
	u, err := s.DB.GetUserByID(ctx, uid)
	if err != nil {
		return "", err
	}
	return u.Plan, nil
}

// Structured limit keys the frontend keys off to show the right upgrade prompt.
const (
	LimitClients   = "clients"
	LimitScenarios = "scenarios"
	LimitTaxYear   = "tax_year"
)

// abortPlanLimit responds with 403 Forbidden and a structured, machine-readable
// error: {"error":"plan_limit","message":...,"limit":...,"upgrade":true}. The
// UI reads `limit` to show the matching upgrade prompt. The limit is always
// enforced here on the server, never by hiding UI alone.
func abortPlanLimit(c *gin.Context, limit, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"error":   "plan_limit",
		"message": msg,
		"limit":   limit,
		"upgrade": true,
	})
}
