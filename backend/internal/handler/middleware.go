package handler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Projects []string `json:"projects"`
	jwt.RegisteredClaims
}

// GenerateJWTToken generates a signed JWT for a user containing their authorized projects
func GenerateJWTToken(jwtSecret string, user *domain.User, projectIDs []string) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Email:    user.Email,
		Projects: projectIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// AuthMiddleware supports both User JWT authentication and Project API Key authentication
func AuthMiddleware(authRepo domain.AuthRepository, jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			apiKeyHeader := c.Request().Header.Get("X-API-Key")

			tokenStr := ""
			if apiKeyHeader != "" {
				tokenStr = apiKeyHeader
			} else if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					tokenStr = parts[1]
				}
			}

			if tokenStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			// Check if API Key format (e.g. starts with "ck_live_")
			if strings.HasPrefix(tokenStr, "ck_live_") {
				// 1. API Key Authentication
				hashBytes := sha256.Sum256([]byte(tokenStr))
				keyHash := fmt.Sprintf("%x", hashBytes)

				apiKey, err := authRepo.GetAPIKeyByHash(c.Request().Context(), keyHash)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to authorize API key"})
				}
				if apiKey == nil {
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired API key"})
				}

				// Update last_used_at asynchronously
				go func(keyID string) {
					_ = authRepo.UpdateAPIKeyLastUsed(context.Background(), keyID)
				}(apiKey.ID)

				c.Set("project_id", apiKey.ProjectID)
				return next(c)
			}

			// 2. User JWT Authentication
			token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired JWT token"})
			}

			claims, ok := token.Claims.(*JWTClaims)
			if !ok || claims.UserID == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
			}

			c.Set("user_id", claims.UserID)

			// Determine target project_id from X-Project-ID header or query param
			projectID := c.Request().Header.Get("X-Project-ID")
			if projectID == "" {
				projectID = c.QueryParam("project_id")
			}

			if projectID != "" {
				// Verify BOLA: Check if projectID is in claims or verify via DB
				isAuthorized := false
				for _, pID := range claims.Projects {
					if pID == projectID {
						isAuthorized = true
						break
					}
				}

				if !isAuthorized {
					// Fallback check against DB
					okDB, dbErr := authRepo.IsUserProjectMemberOrOwner(c.Request().Context(), projectID, claims.UserID)
					if dbErr == nil && okDB {
						isAuthorized = true
					}
				}

				if !isAuthorized {
					return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied: you do not have permission for this project"})
				}

				c.Set("project_id", projectID)
			}

			return next(c)
		}
	}
}

// UsageTrackingMiddleware records API usage asynchronously after each request.
func UsageTrackingMiddleware(usageUsecase domain.UsageUsecase) echo.MiddlewareFunc {
	endpointMap := map[string]string{
		"POST /v1/rooms":                   "rooms.create",
		"GET /v1/rooms/:room_id":           "rooms.get",
		"DELETE /v1/rooms/:room_id":        "rooms.delete",
		"POST /v1/rooms/:room_id/messages": "messages.create",
		"GET /v1/rooms/:room_id/messages":  "messages.list",
		"POST /v1/rooms/:room_id/context":  "context.get",
		"POST /v1/rooms/:room_id/events":   "events.log",
		"GET /v1/rooms/:room_id/events":    "events.stream",
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			projectID, ok := c.Get("project_id").(string)
			if ok && projectID != "" {
				routeKey := c.Request().Method + " " + c.Path()
				endpointName, found := endpointMap[routeKey]
				if found {
					roomID := c.Param("room_id")
					var roomIDPtr *string
					if roomID != "" {
						roomIDPtr = &roomID
					}
					go func() {
						tokensIn, _ := c.Get("tokens_input").(int)
						tokensOut, _ := c.Get("tokens_output").(int)
						if recErr := usageUsecase.RecordUsage(context.Background(), projectID, endpointName, roomIDPtr, tokensIn, tokensOut); recErr != nil {
							log.Printf("[Usage Tracking Error]: %v", recErr)
						}
					}()
				}
			}

			return err
		}
	}
}
