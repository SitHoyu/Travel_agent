package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/travel-agent/shared/contracts"
)

type service interface {
	Generate(context.Context, contracts.LLMGenerateRequest) (contracts.LLMGenerateResponse, error)
	GeneratePlan(context.Context, contracts.GeneratePlanRequest, string, string) (contracts.LLMGenerateResponse, error)
	RevisePlan(context.Context, contracts.RevisePlanRequest, string, string) (contracts.LLMGenerateResponse, error)
}

type Handler struct {
	service service
}

type templateGenerateRequest struct {
	contracts.LLMGenerateRequest
}

type planGenerateRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	contracts.GeneratePlanRequest
}

type revisePlanRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	contracts.RevisePlanRequest
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/healthz", h.healthz)
	v1 := router.Group("/v1")
	v1.POST("/generate", h.generate)
	v1.POST("/travel/plan/generate", h.generatePlan)
	v1.POST("/travel/plan/revise", h.revisePlan)
}

func (h *Handler) healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) generate(ctx *gin.Context) {
	var req templateGenerateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Generate(ctx.Request.Context(), req.LLMGenerateRequest)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) generatePlan(ctx *gin.Context) {
	var req planGenerateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.GeneratePlan(ctx.Request.Context(), req.GeneratePlanRequest, req.Provider, req.Model)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) revisePlan(ctx *gin.Context) {
	var req revisePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RevisePlan(ctx.Request.Context(), req.RevisePlanRequest, req.Provider, req.Model)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
