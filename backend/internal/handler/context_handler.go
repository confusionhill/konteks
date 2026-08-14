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

func (h *ContextHandler) GetContextPackage(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	var req domain.GetContextRequest
	_ = c.Bind(&req)

	if req.RequestKey == "" {
		req.RequestKey = c.Request().Header.Get("X-Encryption-Key")
	}

	pkg, err := h.contextUsecase.GetContextPackage(c.Request().Context(), projectID, roomID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if pkg != nil {
		c.Set("tokens_input", pkg.TokensProcessed)
		c.Set("tokens_output", pkg.TokensEstimated)
	}

	return c.JSON(http.StatusOK, pkg)
}
