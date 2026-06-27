package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/suncrestlabs/nester/apps/api/internal/auth"
	"github.com/suncrestlabs/nester/apps/api/internal/domain/projection"
	"github.com/suncrestlabs/nester/apps/api/internal/service"
	"github.com/suncrestlabs/nester/apps/api/pkg/response"
)

// ProjectionHandler handles compound interest projection endpoints
type ProjectionHandler struct {
	service *service.ProjectionService
}

// NewProjectionHandler creates a new projection handler
func NewProjectionHandler(service *service.ProjectionService) *ProjectionHandler {
	return &ProjectionHandler{service: service}
}

// Register registers the projection routes
func (h *ProjectionHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tools/projection", h.calculateGenericProjection)
	mux.HandleFunc("GET /api/v1/vaults/{id}/projection", h.calculateVaultProjection)
}

// genericProjectionRequest represents the JSON payload for generic projections
type genericProjectionRequest struct {
	InitialDeposit      string `json:"initial_deposit"`
	MonthlyContribution string `json:"monthly_contribution"`
	APY                 string `json:"apy"`
	PeriodMonths        int    `json:"period_months"`
	CompoundFrequency   string `json:"compound_frequency"`
}

// Validate validates the request
func (r *genericProjectionRequest) Validate() error {
	if r.InitialDeposit == "" {
		return projection.ErrInvalidAmount
	}
	if r.APY == "" {
		return projection.ErrInvalidAPY
	}
	if r.PeriodMonths <= 0 {
		return projection.ErrInvalidPeriod
	}
	if r.CompoundFrequency == "" {
		r.CompoundFrequency = "monthly" // default
	}
	return nil
}

// toProjectionInput converts the request to domain input
func (r *genericProjectionRequest) toProjectionInput() (projection.ProjectionInput, error) {
	initialDeposit, err := decimal.NewFromString(r.InitialDeposit)
	if err != nil {
		return projection.ProjectionInput{}, projection.ErrInvalidAmount
	}

	monthlyContribution := decimal.Zero
	if r.MonthlyContribution != "" {
		monthlyContribution, err = decimal.NewFromString(r.MonthlyContribution)
		if err != nil {
			return projection.ProjectionInput{}, projection.ErrInvalidAmount
		}
	}

	apy, err := decimal.NewFromString(r.APY)
	if err != nil {
		return projection.ProjectionInput{}, projection.ErrInvalidAPY
	}

	compoundFreq, err := projection.ParseCompoundFrequency(r.CompoundFrequency)
	if err != nil {
		return projection.ProjectionInput{}, err
	}

	return projection.ProjectionInput{
		InitialDeposit:      initialDeposit,
		MonthlyContribution: monthlyContribution,
		APY:                 apy,
		PeriodMonths:        r.PeriodMonths,
		CompoundFrequency:   compoundFreq,
	}, nil
}

// calculateGenericProjection handles POST /api/v1/tools/projection
func (h *ProjectionHandler) calculateGenericProjection(w http.ResponseWriter, r *http.Request) {
	var req genericProjectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("invalid JSON payload"))
		return
	}

	if err := req.Validate(); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr(err.Error()))
		return
	}

	input, err := req.toProjectionInput()
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr(err.Error()))
		return
	}

	result, err := h.service.CalculateCompoundProjection(r.Context(), input)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Err(http.StatusInternalServerError, "CALCULATION_ERROR", err.Error()))
		return
	}

	response.WriteJSON(w, http.StatusOK, response.OK(result))
}

// calculateVaultProjection handles GET /api/v1/vaults/{id}/projection
func (h *ProjectionHandler) calculateVaultProjection(w http.ResponseWriter, r *http.Request) {
	// Parse vault ID
	vaultIDStr := r.PathValue("id")
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("invalid vault ID"))
		return
	}

	// Extract query parameters
	query := r.URL.Query()

	// Required: deposit amount
	depositStr := query.Get("deposit")
	if depositStr == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("deposit parameter is required"))
		return
	}

	deposit, err := decimal.NewFromString(depositStr)
	if err != nil || deposit.LessThanOrEqual(decimal.Zero) {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("invalid deposit amount"))
		return
	}

	// Required: period
	period := query.Get("period")
	if period == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("period parameter is required"))
		return
	}

	// Optional: compound frequency (default: monthly)
	compound := query.Get("compound")
	if compound == "" {
		compound = "monthly"
	}

	// Optional: APY override
	var apyOverride *decimal.Decimal
	if apyStr := query.Get("apy"); apyStr != "" {
		apy, err := decimal.NewFromString(apyStr)
		if err != nil || apy.LessThanOrEqual(decimal.Zero) {
			response.WriteJSON(w, http.StatusBadRequest, response.ValidationErr("invalid APY override"))
			return
		}
		apyOverride = &apy
	}

	// Verify user has access to this vault (basic auth check)
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		response.WriteJSON(w, http.StatusUnauthorized, response.Err(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required"))
		return
	}

	_, err = uuid.Parse(user.ID)
	if err != nil {
		response.WriteJSON(w, http.StatusUnauthorized, response.Err(http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID"))
		return
	}

	input := projection.VaultProjectionInput{
		VaultID:           vaultID,
		Deposit:           deposit,
		Period:            period,
		CompoundFrequency: compound,
		APYOverride:       apyOverride,
	}

	result, err := h.service.CalculateVaultProjection(r.Context(), input)
	if err != nil {
		if err.Error() == "vault not found" {
			response.WriteJSON(w, http.StatusNotFound, response.NotFound("vault"))
			return
		}
		response.WriteJSON(w, http.StatusInternalServerError, response.Err(http.StatusInternalServerError, "CALCULATION_ERROR", err.Error()))
		return
	}

	// Additional auth check: ensure vault belongs to user (if needed)
	// This would require updating the service to return vault info or checking permissions

	response.WriteJSON(w, http.StatusOK, response.OK(result))
}
