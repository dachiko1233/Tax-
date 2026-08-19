package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ss-tax-engine/internal/db"
	"ss-tax-engine/internal/models"
)

type clientReq struct {
	Name              string  `json:"name"`
	FilingStatus      string  `json:"filing_status"`
	State             string  `json:"state"`
	Age               int     `json:"age"`
	AtFRA             bool    `json:"at_fra"`
	SSBenefits        float64 `json:"ss_benefits"`
	OtherIncome       float64 `json:"other_income"`
	TaxExemptInterest float64 `json:"tax_exempt_interest"`
}

func (r clientReq) validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if !validFilingStatus(r.FilingStatus) {
		return errors.New("invalid filing_status")
	}
	if r.SSBenefits < 0 || r.OtherIncome < 0 || r.TaxExemptInterest < 0 {
		return errors.New("money values must be non-negative")
	}
	return nil
}

func (s *Server) handleListClients(c *gin.Context) {
	clients, err := s.DB.ListClients(context.Background(), userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list clients"})
		return
	}
	if clients == nil {
		clients = []models.Client{}
	}
	c.JSON(http.StatusOK, clients)
}

func (s *Server) handleCreateClient(c *gin.Context) {
	var req clientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := req.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	uid := userID(c)

	// Free-plan limit: at most FreeMaxClients saved clients.
	plan, err := s.planFor(context.Background(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if plan == PlanFree {
		n, err := s.DB.CountClients(context.Background(), uid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		if n >= FreeMaxClients {
			abortPlanLimit(c, LimitClients, "The Free plan is limited to 3 clients. Upgrade to Pro for unlimited clients.")
			return
		}
	}

	cl := reqToClient(req)
	cl.UserID = uid
	saved, err := s.DB.CreateClient(context.Background(), cl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create client"})
		return
	}
	c.JSON(http.StatusCreated, saved)
}

func (s *Server) handleGetClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cl, err := s.DB.GetClient(context.Background(), userID(c), id)
	if err != nil {
		notFoundOr500(c, err)
		return
	}
	c.JSON(http.StatusOK, cl)
}

func (s *Server) handleUpdateClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req clientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := req.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cl := reqToClient(req)
	cl.ID = id
	updated, err := s.DB.UpdateClient(context.Background(), userID(c), cl)
	if err != nil {
		notFoundOr500(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleDeleteClient(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.DB.DeleteClient(context.Background(), userID(c), id); err != nil {
		notFoundOr500(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func reqToClient(r clientReq) *models.Client {
	return &models.Client{
		Name: r.Name, FilingStatus: r.FilingStatus, State: r.State,
		Age: r.Age, AtFRA: r.AtFRA, SSBenefits: r.SSBenefits,
		OtherIncome: r.OtherIncome, TaxExemptInterest: r.TaxExemptInterest,
	}
}

func notFoundOr500(c *gin.Context, err error) {
	if errors.Is(err, db.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
}
