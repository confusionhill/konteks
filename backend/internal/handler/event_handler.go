package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type EventBroker struct {
	subscribers map[string][]chan string
	mu          sync.RWMutex
}

func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[string][]chan string),
	}
}

func (b *EventBroker) Subscribe(roomID string) chan string {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 100) // Buffering to avoid hanging on slow network clients
	b.subscribers[roomID] = append(b.subscribers[roomID], ch)
	return ch
}

func (b *EventBroker) Unsubscribe(roomID string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[roomID]
	for i, sub := range subs {
		if sub == ch {
			close(ch)
			b.subscribers[roomID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (b *EventBroker) Publish(roomID string, message string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[roomID]
	if !ok {
		return
	}

	for _, sub := range subs {
		select {
		case sub <- message:
		default:
			// Client channel full, skip to avoid slow consumer blocking
		}
	}
}

type EventHandler struct {
	eventUsecase domain.EventUsecase
	broker       *EventBroker
}

func NewEventHandler(eventUsecase domain.EventUsecase, broker *EventBroker) *EventHandler {
	return &EventHandler{
		eventUsecase: eventUsecase,
		broker:       broker,
	}
}

type logEventRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (h *EventHandler) LogEvent(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	var req logEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event type cannot be empty"})
	}

	event, err := h.eventUsecase.LogEvent(c.Request().Context(), tenantID, roomID, req.Type, req.Payload)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Broadcast via SSE in real time
	eventJSON, err := json.Marshal(event)
	if err == nil {
		h.broker.Publish(roomID, string(eventJSON))
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "logged"})
}

func (h *EventHandler) StreamEvents(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id parameter"})
	}

	// Double check room exists for this tenant before subscribing
	_, err := h.eventUsecase.ListEvents(c.Request().Context(), tenantID, roomID, 1)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "room not found or unauthorized"})
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no")

	ch := h.broker.Subscribe(roomID)
	defer h.broker.Unsubscribe(roomID, ch)

	// Keep alive ticker to prevent connection termination
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if _, err := fmt.Fprintf(c.Response().Writer, "data: %s\n\n", msg); err != nil {
				return nil
			}
			c.Response().Flush()
		case <-c.Request().Context().Done():
			return nil
		}
	}
}
