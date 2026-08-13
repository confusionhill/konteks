package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	usageUsecase   domain.UsageUsecase
	roomUsecase    domain.RoomUsecase
	messageUsecase domain.MessageUsecase
	roomRepo       domain.RoomRepository
	storage        domain.StorageProvider
	evalRepo       domain.EvaluationRepository
}

func NewDashboardHandler(
	usageUsecase domain.UsageUsecase,
	roomUsecase domain.RoomUsecase,
	messageUsecase domain.MessageUsecase,
	roomRepo domain.RoomRepository,
	storage domain.StorageProvider,
	evalRepo domain.EvaluationRepository,
) *DashboardHandler {
	return &DashboardHandler{
		usageUsecase:   usageUsecase,
		roomUsecase:    roomUsecase,
		messageUsecase: messageUsecase,
		roomRepo:       roomRepo,
		storage:        storage,
		evalRepo:       evalRepo,
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

	search := c.QueryParam("search")
	status := c.QueryParam("status")
	sortBy := c.QueryParam("sort_by")
	sortOrder := c.QueryParam("sort_order")

	limitStr := c.QueryParam("limit")
	limit := 10
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	pageStr := c.QueryParam("page")
	page := 1
	if pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			page = val
		}
	}
	offset := (page - 1) * limit

	minMessagesStr := c.QueryParam("min_messages")
	minMessages := 0
	if minMessagesStr != "" {
		if val, err := strconv.Atoi(minMessagesStr); err == nil && val >= 0 {
			minMessages = val
		}
	}

	rooms, total, err := h.roomUsecase.ListRoomsOverview(
		c.Request().Context(), tenantID, search, status, sortBy, sortOrder, minMessages, limit, offset,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"rooms": rooms,
		"total": total,
	})
}

type roomDetailResponse struct {
	Room     *domain.Room        `json:"room"`
	Metrics  *domain.RoomMetrics `json:"metrics"`
	State    any                 `json:"state"`
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

	var messages []domain.Message
	if room.Status != domain.RoomStatusArchived {
		messages, err = h.messageUsecase.ListMessages(c.Request().Context(), tenantID, roomID, 50, 0)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	} else {
		messages = []domain.Message{}
	}

	metrics, err := h.usageUsecase.GetRoomMetrics(c.Request().Context(), roomID)
	if err != nil {
		metrics = &domain.RoomMetrics{}
	}

	// Clean zero-cost read-only state response directly from PostgreSQL DB
	var stateResp any
	if state != nil {
		var parsedState any
		if err := json.Unmarshal(state.State, &parsedState); err != nil {
			parsedState = make(map[string]any)
		}

		stateResp = map[string]interface{}{
			"room_id":      state.RoomID,
			"summary":      state.Summary,
			"active_topic": state.ActiveTopic,
			"state":        parsedState,
			"updated_at":   state.UpdatedAt,
			"references":   state.References,
			"entities":     state.Entities,
		}
	}

	return c.JSON(http.StatusOK, roomDetailResponse{
		Room:     room,
		Metrics:  metrics,
		State:    stateResp,
		Messages: messages,
	})
}

func (h *DashboardHandler) GetArchiveURL(c echo.Context) error {
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
	if room.Status != domain.RoomStatusArchived || room.ArchiveS3Path == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room is not in cold storage"})
	}

	// 15 minute lifetime
	url, err := h.storage.GeneratePresignedURL(c.Request().Context(), *room.ArchiveS3Path, 15 * 60 * 1000 * 1000 * 1000)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

func (h *DashboardHandler) GetEvaluationsSummary(c echo.Context) error {
	summary, err := h.evalRepo.GetSummary(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, summary)
}

func (h *DashboardHandler) GetRoomEvaluations(c echo.Context) error {
	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id"})
	}

	evals, err := h.evalRepo.GetByRoomID(c.Request().Context(), roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if evals == nil {
		evals = []domain.ContextEvaluation{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"evaluations": evals,
	})
}
