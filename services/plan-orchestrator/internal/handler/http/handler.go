package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/travel-agent/shared/contracts"
)

type service interface {
	RunPlan(context.Context, contracts.GeneratePlanRequest) (contracts.AgentPlanResponse, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/healthz", h.healthz)
	v1 := router.Group("/v1")
	v1.POST("/agent/plan/run", h.runPlan)
}

func (h *Handler) healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) runPlan(ctx *gin.Context) {
	var req contracts.GeneratePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.RunPlan(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
