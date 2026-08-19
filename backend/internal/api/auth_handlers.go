package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"ss-tax-engine/internal/auth"
	"ss-tax-engine/internal/models"
)

// verificationTTL is how long an email-verification link stays valid.
const verificationTTL = 24 * time.Hour

// resendWindow / resendMax rate-limit the "resend verification" endpoint.
const (
	resendWindow = time.Hour
	resendMax    = 3
)

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FirmName string `json:"firm_name"`
}

type authResp struct {
	Token         string `json:"token"`
	Email         string `json:"email"`
	FirmName      string `json:"firm_name"`
	Plan          string `json:"plan"`
	EmailVerified bool   `json:"email_verified"`
}

func (s *Server) handleRegister(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(req.Email, "@") || len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid email and password (min 8 chars) required"})
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hashing failed"})
		return
	}
	u, err := s.DB.CreateUser(context.Background(), req.Email, hash, req.FirmName)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}

	devLink := s.issueAndSendVerification(u)

	resp := gin.H{
		"email":                 u.Email,
		"verification_required": true,
		"message":               "Account created. Check your email for a verification link before signing in.",
	}
	if devLink != "" { // only populated when email delivery is not configured
		resp["dev_verify_link"] = devLink
	}
	c.JSON(http.StatusCreated, resp)
}

func (s *Server) handleLogin(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.DB.GetUserByEmail(context.Background(), req.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !u.EmailVerified {
		c.JSON(http.StatusForbidden, gin.H{
			"error":                 "email not verified",
			"verification_required": true,
		})
		return
	}
	token, err := s.Auth.Issue(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue token"})
		return
	}
	c.JSON(http.StatusOK, authResp{
		Token: token, Email: u.Email, FirmName: u.FirmName,
		Plan: u.Plan, EmailVerified: true,
	})
}

// handleVerifyEmail consumes a verification token and marks the account verified.
func (s *Server) handleVerifyEmail(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("token"))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}
	userID, err := s.DB.ConsumeVerificationToken(context.Background(), auth.HashToken(raw))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}
	if err := s.DB.SetEmailVerified(context.Background(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not verify"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email verified. You can now sign in."})
}

// handleResendVerification re-issues a verification email, rate-limited per user.
// It always returns 200 for unknown/verified emails so it can't be used to probe
// which addresses have accounts.
func (s *Server) handleResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	ok := gin.H{"message": "If that account exists and is unverified, a new link has been sent."}

	u, err := s.DB.GetUserByEmail(context.Background(), req.Email)
	if err != nil || u.EmailVerified {
		c.JSON(http.StatusOK, ok)
		return
	}
	n, err := s.DB.RecentTokenCount(context.Background(), u.ID, time.Now().Add(-resendWindow))
	if err == nil && n >= resendMax {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests; try again later"})
		return
	}
	s.issueAndSendVerification(u)
	c.JSON(http.StatusOK, ok)
}

// issueAndSendVerification creates a token, persists its hash, and sends (or in
// dev mode logs) the verification email. It returns a non-empty link only when
// email delivery is NOT configured, so callers can surface it for local dev.
func (s *Server) issueAndSendVerification(u *models.User) string {
	raw, hash, err := auth.GenerateToken()
	if err != nil {
		return ""
	}
	if err := s.DB.CreateVerificationToken(context.Background(), u.ID, hash, time.Now().Add(verificationTTL)); err != nil {
		return ""
	}
	_ = s.Mail.SendVerification(context.Background(), u.Email, raw)
	if !s.Mail.Enabled() {
		return s.Mail.VerifyLink(raw)
	}
	return ""
}
