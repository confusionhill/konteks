package handler

import (
	"net/http"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type ContextHealthHandler struct {
	healthUsecase domain.ContextHealthUsecase
}

func NewContextHealthHandler(healthUsecase domain.ContextHealthUsecase) *ContextHealthHandler {
	return &ContextHealthHandler{
		healthUsecase: healthUsecase,
	}
}

func (h *ContextHealthHandler) GetPolicy(c echo.Context) error {
	projectID := c.Param("project_id")
	if projectID == "" {
		projectID, _ = c.Get("project_id").(string)
	}
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id is required"})
	}

	policy, err := h.healthUsecase.GetProjectPolicy(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, policy)
}

func (h *ContextHealthHandler) UpdatePolicy(c echo.Context) error {
	projectID := c.Param("project_id")
	if projectID == "" {
		projectID, _ = c.Get("project_id").(string)
	}
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id is required"})
	}

	var policy domain.ContextHealthPolicy
	if err := c.Bind(&policy); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
	}

	if err := h.healthUsecase.UpdateProjectPolicy(c.Request().Context(), projectID, &policy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, policy)
}

func (h *ContextHealthHandler) ListRoomMemories(c echo.Context) error {
	projectID, _ := c.Get("project_id").(string)
	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id is required"})
	}

	includeSuperseded := c.QueryParam("include_superseded") == "true"

	memories, err := h.healthUsecase.ListRoomMemories(c.Request().Context(), projectID, roomID, includeSuperseded)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"memories": memories,
		"count":    len(memories),
	})
}

func (h *ContextHealthHandler) RestoreMemory(c echo.Context) error {
	projectID, _ := c.Get("project_id").(string)
	memoryID := c.Param("memory_id")
	if memoryID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "memory_id is required"})
	}

	if err := h.healthUsecase.RestoreMemory(c.Request().Context(), projectID, memoryID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":    "restored",
		"memory_id": memoryID,
	})
}

func (h *ContextHealthHandler) CleanRoom(c echo.Context) error {
	projectID, _ := c.Get("project_id").(string)
	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id is required"})
	}

	if err := h.healthUsecase.CleanRoomContextHealth(c.Request().Context(), projectID, roomID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"status":  "cleanup_completed",
		"room_id": roomID,
	})
}
