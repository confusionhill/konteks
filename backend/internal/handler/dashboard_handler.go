package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	summary, err := h.usageUsecase.GetUsageSummary(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, summary)
}

func (h *DashboardHandler) ListRooms(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
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
		c.Request().Context(), projectID, search, status, sortBy, sortOrder, minMessages, limit, offset,
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
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id"})
	}

	room, err := h.roomRepo.GetByID(c.Request().Context(), projectID, roomID)
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
		messages, err = h.messageUsecase.ListMessages(c.Request().Context(), projectID, roomID, 50, 0)
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
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing room_id"})
	}

	room, err := h.roomRepo.GetByID(c.Request().Context(), projectID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if room == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "room not found"})
	}
	if room.Status != domain.RoomStatusArchived || room.ArchiveS3Path == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room is not in cold storage"})
	}

	url, err := h.storage.GeneratePresignedURL(c.Request().Context(), *room.ArchiveS3Path, 15*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

func (h *DashboardHandler) GetEvaluationsSummary(c echo.Context) error {
	projectID, _ := c.Get("project_id").(string)

	summary, err := h.evalRepo.GetSummary(c.Request().Context(), projectID)
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

func (h *DashboardHandler) GetProjectLogs(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	roomID := c.QueryParam("room_id")
	externalUserID := c.QueryParam("external_user_id")
	search := c.QueryParam("search")
	role := c.QueryParam("role")
	sortBy := c.QueryParam("sort_by")
	sortOrder := c.QueryParam("sort_order")

	limitStr := c.QueryParam("limit")
	limit := 20
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

	logs, total, err := h.messageUsecase.ListProjectLogs(
		c.Request().Context(), projectID, roomID, externalUserID, search, role, sortBy, sortOrder, limit, offset,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *DashboardHandler) GetProjectEvaluations(c echo.Context) error {
	projectID, ok := c.Get("project_id").(string)
	if !ok || projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project context required"})
	}

	limitStr := c.QueryParam("limit")
	limit := 20
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

	evals, total, err := h.evalRepo.GetByProjectID(c.Request().Context(), projectID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if evals == nil {
		evals = []domain.ContextEvaluation{}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"evaluations": evals,
		"total":       total,
		"page":        page,
		"limit":       limit,
	})
}

func (h *DashboardHandler) GetEvaluationDetail(c echo.Context) error {
	evalID := c.Param("eval_id")
	if evalID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing eval_id"})
	}

	eval, err := h.evalRepo.GetByID(c.Request().Context(), evalID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "evaluation not found"})
	}

	return c.JSON(http.StatusOK, eval)
}
