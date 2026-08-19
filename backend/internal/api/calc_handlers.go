package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ss-tax-engine/internal/engine"
	"ss-tax-engine/internal/models"
)

type calcReq struct {
	FilingStatus      string  `json:"filing_status"`
	State             string  `json:"state"`
	Age               int     `json:"age"`
	AtFRA             bool    `json:"at_fra"`
	SSBenefits        float64 `json:"ss_benefits"`
	OtherIncome       float64 `json:"other_income"`
	TaxExemptInterest float64 `json:"tax_exempt_interest"`
	TaxYear           int     `json:"tax_year"`
}

func (r calcReq) toInput() engine.Input {
	year := r.TaxYear
	if year == 0 {
		year = 2026
	}
	return engine.Input{
		FilingStatus:      engine.FilingStatus(r.FilingStatus),
		State:             r.State,
		Age:               r.Age,
		AtFRA:             r.AtFRA,
		SSBenefits:        r.SSBenefits,
		OtherIncome:       r.OtherIncome,
		TaxExemptInterest: r.TaxExemptInterest,
		TaxYear:           year,
	}
}

// handleCalculate runs the engine on the backend — never trusting any numbers
// computed client-side. The engine is the source of truth.
func (s *Server) handleCalculate(c *gin.Context) {
	var req calcReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	in := req.toInput()

	// Free plan is limited to the current tax year; multi-year what-if is Pro.
	plan, err := s.planFor(context.Background(), userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if plan == PlanFree && in.TaxYear != CurrentTaxYear {
		abortPlanLimit(c, LimitTaxYear, "The Free plan supports the current tax year only. Upgrade to Pro for multi-year planning.")
		return
	}

	res, err := s.Engine.Calculate(in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

type scenarioReq struct {
	Label  string  `json:"label"`
	Inputs calcReq `json:"inputs"`
}

func (s *Server) handleCreateScenario(c *gin.Context) {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	owned, err := s.DB.ClientOwnedBy(context.Background(), userID(c), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if !owned {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	var req scenarioReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	in := req.Inputs.toInput()

	// Free-plan limits: current tax year only, and one saved scenario per client.
	plan, err := s.planFor(context.Background(), userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if plan == PlanFree {
		if in.TaxYear != CurrentTaxYear {
			abortPlanLimit(c, LimitTaxYear, "The Free plan supports the current tax year only. Upgrade to Pro for multi-year planning.")
			return
		}
		n, err := s.DB.CountScenarios(context.Background(), clientID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		if n >= FreeMaxScenariosPerClnt {
			abortPlanLimit(c, LimitScenarios, "The Free plan allows one saved scenario per client. Upgrade to Pro for unlimited scenarios.")
			return
		}
	}

	res, err := s.Engine.Calculate(in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sc := &models.Scenario{
		ClientID:    clientID,
		TaxYear:     in.TaxYear,
		Label:       req.Label,
		InputsJSON:  toMap(in),
		ResultsJSON: toMap(res),
	}
	saved, err := s.DB.CreateScenario(context.Background(), sc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save scenario"})
		return
	}
	c.JSON(http.StatusCreated, saved)
}

func (s *Server) handleListScenarios(c *gin.Context) {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	owned, err := s.DB.ClientOwnedBy(context.Background(), userID(c), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if !owned {
		c.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}
	list, err := s.DB.ListScenarios(context.Background(), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list scenarios"})
		return
	}
	if list == nil {
		list = []models.Scenario{}
	}
	c.JSON(http.StatusOK, list)
}

// toMap round-trips a struct through JSON into a generic map for JSONB storage.
func toMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	return m
}

func validFilingStatus(s string) bool {
	for _, v := range engine.ValidFilingStatuses {
		if string(v) == s {
			return true
		}
	}
	return false
}
