package handler

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

func AuthMiddleware(authRepo domain.AuthRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format, must be Bearer <token>"})
			}

			token := parts[1]
			// Hash the token using SHA-256 to compare with db record
			hashBytes := sha256.Sum256([]byte(token))
			keyHash := fmt.Sprintf("%x", hashBytes)

			tenant, err := authRepo.GetTenantByAPIKey(c.Request().Context(), keyHash)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to authorize request"})
			}

			if tenant == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			}

			// Store tenant ID in context for downstream handlers
			c.Set("tenant_id", tenant.ID)
			return next(c)
		}
	}
}
