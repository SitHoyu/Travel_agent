package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/travel-agent/services/plan-orchestrator/internal/middleware"
	"github.com/travel-agent/shared/contracts"
)

type planService interface {
	GeneratePlan(context.Context, contracts.GeneratePlanRequest) (contracts.AgentPlanResponse, error)
	SavePlan(context.Context, contracts.SavePlanRequest) (contracts.SavedPlanResponse, error)
	ListPlans(context.Context, int64, int, int) (contracts.ListPlansResponse, error)
	GetPlan(context.Context, int64, int64) (contracts.SavedPlanResponse, bool, error)
}

type authService interface {
	Register(context.Context, contracts.RegisterRequest) (contracts.AuthResponse, error)
	Login(context.Context, contracts.LoginRequest) (contracts.AuthResponse, error)
	GetCurrentUser(context.Context, int64) (contracts.UserProfile, error)
}

type Handler struct {
	planService    planService
	authService    authService
	authMiddleware gin.HandlerFunc
}

func NewHandler(planService planService, authService authService, authMiddleware gin.HandlerFunc) *Handler {
	return &Handler{
		planService:    planService,
		authService:    authService,
		authMiddleware: authMiddleware,
	}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/healthz", h.healthz)
	v1 := router.Group("/v1")

	authGroup := v1.Group("/auth")
	authGroup.POST("/register", h.register)
	authGroup.POST("/login", h.login)

	protected := v1.Group("/")
	protected.Use(h.authMiddleware)
	protected.GET("/users/me", h.me)
	protected.POST("/agent/plan/run", h.runPlan)
	protected.POST("/plans", h.savePlan)
	protected.GET("/plans", h.listPlans)
	protected.GET("/plans/:id", h.getPlan)
}

func (h *Handler) healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) register(ctx *gin.Context) {
	var req contracts.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Register(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) login(ctx *gin.Context) {
	var req contracts.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) me(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	resp, err := h.authService.GetCurrentUser(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) runPlan(ctx *gin.Context) {
	var req contracts.GeneratePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.planService.GeneratePlan(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) savePlan(ctx *gin.Context) {
	var req contracts.SavePlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}
	req.UserID = userID

	resp, err := h.planService.SavePlan(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) listPlans(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	page := parseOptionalInt(ctx.DefaultQuery("page", "1"), 1)
	pageSize := parseOptionalInt(ctx.DefaultQuery("page_size", "20"), 20)

	resp, err := h.planService.ListPlans(ctx.Request.Context(), userID, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (h *Handler) getPlan(ctx *gin.Context) {
	userID, ok := currentUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	planID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || planID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id must be a positive integer"})
		return
	}

	resp, found, err := h.planService.GetPlan(ctx.Request.Context(), userID, planID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !found {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func parseOptionalInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func currentUserID(ctx *gin.Context) (int64, bool) {
	value, ok := ctx.Get(middleware.ContextUserIDKey)
	if !ok {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok
}
