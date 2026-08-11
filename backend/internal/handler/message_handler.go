package handler

import (
	"net/http"
	"strconv"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type MessageHandler struct {
	messageUsecase domain.MessageUsecase
}

func NewMessageHandler(messageUsecase domain.MessageUsecase) *MessageHandler {
	return &MessageHandler{messageUsecase: messageUsecase}
}

type createMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *MessageHandler) CreateMessage(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	var req createMessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Role != "user" && req.Role != "assistant" && req.Role != "system" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be user, assistant, or system"})
	}
	if req.Content == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content cannot be empty"})
	}

	msg, err := h.messageUsecase.CreateMessage(c.Request().Context(), tenantID, roomID, req.Role, req.Content)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, msg)
}

func (h *MessageHandler) ListMessages(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	limitStr := c.QueryParam("limit")
	limit := 20
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	beforeSeqStr := c.QueryParam("before_sequence")
	var beforeSeq int64 = 0
	if beforeSeqStr != "" {
		if val, err := strconv.ParseInt(beforeSeqStr, 10, 64); err == nil {
			beforeSeq = val
		}
	}

	messages, err := h.messageUsecase.ListMessages(c.Request().Context(), tenantID, roomID, limit, beforeSeq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"messages": messages,
		"has_more": len(messages) >= limit,
	})
}
