package handler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
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

// UsageTrackingMiddleware records API usage asynchronously after each request.
func UsageTrackingMiddleware(usageUsecase domain.UsageUsecase) echo.MiddlewareFunc {
	// Map Echo routes to readable endpoint names
	endpointMap := map[string]string{
		"POST /v1/rooms":                     "rooms.create",
		"GET /v1/rooms/:room_id":             "rooms.get",
		"DELETE /v1/rooms/:room_id":          "rooms.delete",
		"POST /v1/rooms/:room_id/messages":   "messages.create",
		"GET /v1/rooms/:room_id/messages":    "messages.list",
		"POST /v1/rooms/:room_id/context":    "context.get",
		"POST /v1/rooms/:room_id/events":     "events.log",
		"GET /v1/rooms/:room_id/events":      "events.stream",
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Execute the handler first
			err := next(c)

			// Fire-and-forget usage recording after response
			tenantID, ok := c.Get("tenant_id").(string)
			if ok && tenantID != "" {
				routeKey := c.Request().Method + " " + c.Path()
				endpointName, found := endpointMap[routeKey]
				if found {
					roomID := c.Param("room_id")
					var roomIDPtr *string
					if roomID != "" {
						roomIDPtr = &roomID
					}
					go func() {
						if recErr := usageUsecase.RecordUsage(context.Background(), tenantID, endpointName, roomIDPtr, 0); recErr != nil {
							log.Printf("[Usage Tracking Error]: %v", recErr)
						}
					}()
				}
			}

			return err
		}
	}
}
