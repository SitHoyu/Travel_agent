package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	authutil "github.com/travel-agent/services/plan-orchestrator/internal/auth"
)

const (
	ContextUserIDKey   = "current_user_id"
	ContextUsernameKey = "current_username"
)

func AuthRequired(tokenManager *authutil.TokenManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := strings.TrimSpace(ctx.GetHeader("Authorization"))
		if header == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		tokenString, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(tokenString) == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		claims, err := tokenManager.Parse(strings.TrimSpace(tokenString))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		ctx.Set(ContextUserIDKey, claims.UserID)
		ctx.Set(ContextUsernameKey, claims.Username)
		ctx.Next()
	}
}
