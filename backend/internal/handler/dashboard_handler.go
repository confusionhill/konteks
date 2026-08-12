package handler

import (
	"net/http"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	usageUsecase   domain.UsageUsecase
	roomUsecase    domain.RoomUsecase
	messageUsecase domain.MessageUsecase
	roomRepo       domain.RoomRepository
}

func NewDashboardHandler(
	usageUsecase domain.UsageUsecase,
	roomUsecase domain.RoomUsecase,
	messageUsecase domain.MessageUsecase,
	roomRepo domain.RoomRepository,
) *DashboardHandler {
	return &DashboardHandler{
		usageUsecase:   usageUsecase,
		roomUsecase:    roomUsecase,
		messageUsecase: messageUsecase,
		roomRepo:       roomRepo,
	}
}

func (h *DashboardHandler) GetSummary(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	summary, err := h.usageUsecase.GetUsageSummary(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, summary)
}

func (h *DashboardHandler) ListRooms(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	rooms, err := h.roomUsecase.ListRoomsOverview(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, rooms)
}

type roomDetailResponse struct {
	Room     *domain.Room        `json:"room"`
	Metrics  *domain.RoomMetrics `json:"metrics"`
	State    *domain.RoomState   `json:"state"`
	Messages []domain.Message    `json:"messages"`
}

func (h *DashboardHandler) GetRoomDetail(c echo.Context) error {
	tenantID, ok := c.Get("tenant_id").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "tenant context not found"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id"})
	}

	room, err := h.roomRepo.GetByID(c.Request().Context(), tenantID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if room == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "room not found"})
	}

	state, err := h.roomRepo.GetState(c.Request().Context(), roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	messages, err := h.messageUsecase.ListMessages(c.Request().Context(), tenantID, roomID, 50, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	metrics, err := h.usageUsecase.GetRoomMetrics(c.Request().Context(), roomID)
	if err != nil {
		metrics = &domain.RoomMetrics{}
	}

	return c.JSON(http.StatusOK, roomDetailResponse{
		Room:     room,
		Metrics:  metrics,
		State:    state,
		Messages: messages,
	})
}
