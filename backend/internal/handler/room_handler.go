package handler

import (
	"encoding/json"
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
	ExternalUserID string                `json:"external_user_id"`
	Description    string                `json:"description,omitempty"`
	ParentRoomID   *string               `json:"parent_room_id,omitempty"`
	StateSchema    json.RawMessage       `json:"state_schema,omitempty"`
	DefaultPolicy  *domain.ContextPolicy `json:"default_policy,omitempty"`
}

func (h *RoomHandler) CreateRoom(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	var req createRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	room, err := h.roomUsecase.CreateRoom(c.Request().Context(), projectID, req.ExternalUserID, req.Description, req.ParentRoomID, req.StateSchema, req.DefaultPolicy)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) GetRoomState(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	state, err := h.roomUsecase.GetRoomState(c.Request().Context(), projectID, roomID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, state)
}

func (h *RoomHandler) GetChildRooms(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	children, err := h.roomUsecase.ListChildRooms(c.Request().Context(), projectID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{"children": children})
}

func (h *RoomHandler) DeleteRoom(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	err := h.roomUsecase.DeleteRoom(c.Request().Context(), projectID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

type updateRoomRequest struct {
	Description   *string               `json:"description,omitempty"`
	ParentRoomID  *string               `json:"parent_room_id,omitempty"`
	StateSchema   json.RawMessage       `json:"state_schema,omitempty"`
	DefaultPolicy *domain.ContextPolicy `json:"default_policy,omitempty"`
}

func (h *RoomHandler) UpdateRoom(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	var req updateRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	room, err := h.roomUsecase.UpdateRoom(c.Request().Context(), projectID, roomID, req.Description, req.ParentRoomID, req.StateSchema, req.DefaultPolicy)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, room)
}
