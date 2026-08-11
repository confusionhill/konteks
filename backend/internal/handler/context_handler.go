package handler

import (
	"net/http"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type ContextHandler struct {
	contextUsecase domain.ContextUsecase
}

func NewContextHandler(contextUsecase domain.ContextUsecase) *ContextHandler {
	return &ContextHandler{contextUsecase: contextUsecase}
}

type getContextRequest struct {
	Query     string `json:"query"`
	MaxTokens int    `json:"max_tokens"`
}

func (h *ContextHandler) GetContextPackage(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	var req getContextRequest
	// For context fetch, body is optional. Bind if provided
	_ = c.Bind(&req)

	if req.MaxTokens <= 0 {
		req.MaxTokens = 4000 // Default token budget
	}

	pkg, err := h.contextUsecase.GetContextPackage(c.Request().Context(), tenantID, roomID, req.Query, req.MaxTokens)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, pkg)
}
