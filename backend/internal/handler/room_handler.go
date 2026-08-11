package handler

import (
	"net/http"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type RoomHandler struct {
	roomUsecase domain.RoomUsecase
}

func NewRoomHandler(roomUsecase domain.RoomUsecase) *RoomHandler {
	return &RoomHandler{roomUsecase: roomUsecase}
}

type createRoomRequest struct {
	ExternalUserID string `json:"external_user_id"`
}

func (h *RoomHandler) CreateRoom(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	var req createRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	room, err := h.roomUsecase.CreateRoom(c.Request().Context(), tenantID, req.ExternalUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) GetRoomState(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	state, err := h.roomUsecase.GetRoomState(c.Request().Context(), tenantID, roomID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	err := h.roomUsecase.DeleteRoom(c.Request().Context(), tenantID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}
