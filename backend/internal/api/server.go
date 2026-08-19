// Package api wires HTTP routes (Gin) to the auth, db, and engine layers.
package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"ss-tax-engine/internal/auth"
	"ss-tax-engine/internal/billing"
	"ss-tax-engine/internal/db"
	"ss-tax-engine/internal/email"
	"ss-tax-engine/internal/engine"
)

// Plan tiers and their Free-plan limits, enforced server-side.
const (
	PlanFree = "free"
	PlanPro  = "pro"

	// CurrentTaxYear is the only year Free users may compute/save against.
	CurrentTaxYear = 2026

	FreeMaxClients          = 3
	FreeMaxScenariosPerClnt = 1
)

type Server struct {
	DB         *db.DB
	Auth       *auth.Auth
	Engine     *engine.Engine
	Mail       *email.Mailer
	Bill       *billing.Client
	AppBaseURL string
}

func NewServer(database *db.DB, a *auth.Auth, e *engine.Engine, m *email.Mailer, b *billing.Client, appBaseURL string) *Server {
	return &Server{DB: database, Auth: a, Engine: e, Mail: m, Bill: b, AppBaseURL: appBaseURL}
}

// Router builds the Gin engine with all routes registered.
func (s *Server) Router() *gin.Engine {
	r := gin.Default()

	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(corsCfg))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	api := r.Group("/api")
	{
		api.POST("/auth/register", s.handleRegister)
		api.POST("/auth/login", s.handleLogin)
		api.GET("/auth/verify", s.handleVerifyEmail)
		api.POST("/auth/resend", s.handleResendVerification)

		// Dodo posts here server-to-server; it authenticates via signature, not JWT.
		api.POST("/webhooks/dodo", s.handleDodoWebhook)

		authed := api.Group("")
		authed.Use(s.requireAuth())
		{
			authed.GET("/clients", s.handleListClients)
			authed.POST("/clients", s.handleCreateClient)
			authed.GET("/clients/:id", s.handleGetClient)
			authed.PUT("/clients/:id", s.handleUpdateClient)
			authed.DELETE("/clients/:id", s.handleDeleteClient)

			authed.POST("/calculate", s.handleCalculate)

			authed.POST("/clients/:id/scenarios", s.handleCreateScenario)
			authed.GET("/clients/:id/scenarios", s.handleListScenarios)

			authed.POST("/billing/checkout", s.handleCheckout)
			authed.GET("/billing/status", s.handleBillingStatus)
			authed.POST("/billing/verify", s.handleBillingVerify)
		}
	}
	return r
}
