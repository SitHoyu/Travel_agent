package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := strings.TrimSpace(ctx.GetHeader("Origin"))
		if origin != "" && isAllowedOrigin(origin) {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			ctx.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "http://[::1]:")
}
