package handler

import (
	"net/http"
	"strconv"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type ArtifactHandler struct {
	artifactUsecase domain.ArtifactUsecase
}

func NewArtifactHandler(artifactUsecase domain.ArtifactUsecase) *ArtifactHandler {
	return &ArtifactHandler{artifactUsecase: artifactUsecase}
}

func getProjectID(c echo.Context) string {
	if p, ok := c.Get("project_id").(string); ok && p != "" {
		return p
	}
	return c.Param("project_id")
}

func (h *ArtifactHandler) CreateArtifact(c echo.Context) error {
	projectID := getProjectID(c)
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id parameter or header required"})
	}

	var req domain.CreateArtifactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	art, err := h.artifactUsecase.IngestArtifact(c.Request().Context(), projectID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, art)
}

func (h *ArtifactHandler) UpdateArtifact(c echo.Context) error {
	projectID := getProjectID(c)
	artifactID := c.Param("artifact_id")

	if projectID == "" || artifactID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id and artifact_id parameters required"})
	}

	var req domain.UpdateArtifactRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	art, err := h.artifactUsecase.UpdateArtifact(c.Request().Context(), projectID, artifactID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, art)
}

func (h *ArtifactHandler) GetArtifact(c echo.Context) error {
	projectID := getProjectID(c)
	artifactID := c.Param("artifact_id")

	if projectID == "" || artifactID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id and artifact_id parameters required"})
	}

	art, err := h.artifactUsecase.GetArtifact(c.Request().Context(), projectID, artifactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, art)
}

func (h *ArtifactHandler) ListArtifacts(c echo.Context) error {
	projectID := getProjectID(c)
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id parameter required"})
	}

	groupName := c.QueryParam("group_name")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	if limit <= 0 {
		limit = 50
	}

	artifacts, total, err := h.artifactUsecase.ListArtifacts(c.Request().Context(), projectID, groupName, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"artifacts": artifacts,
	})
}

func (h *ArtifactHandler) DeleteArtifact(c echo.Context) error {
	projectID := getProjectID(c)
	artifactID := c.Param("artifact_id")

	if projectID == "" || artifactID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id and artifact_id parameters required"})
	}

	err := h.artifactUsecase.DeleteArtifact(c.Request().Context(), projectID, artifactID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ArtifactHandler) GetKnowledgeGraph(c echo.Context) error {
	projectID := getProjectID(c)
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id parameter required"})
	}

	graph, err := h.artifactUsecase.GetKnowledgeGraph(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, graph)
}

func (h *ArtifactHandler) AddRoomBlacklist(c echo.Context) error {
	projectID := getProjectID(c)
	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id parameter required"})
	}

	var req domain.AddRoomBlacklistRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	bl, err := h.artifactUsecase.AddRoomBlacklist(c.Request().Context(), projectID, roomID, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, bl)
}

func (h *ArtifactHandler) RemoveRoomBlacklist(c echo.Context) error {
	projectID := getProjectID(c)
	roomID := c.Param("room_id")
	blacklistID := c.Param("blacklist_id")

	if roomID == "" || blacklistID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id and blacklist_id parameters required"})
	}

	err := h.artifactUsecase.RemoveRoomBlacklist(c.Request().Context(), projectID, roomID, blacklistID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ArtifactHandler) GetRoomBlacklists(c echo.Context) error {
	projectID := getProjectID(c)
	roomID := c.Param("room_id")
	if roomID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "room_id parameter required"})
	}

	blacklists, err := h.artifactUsecase.GetRoomBlacklists(c.Request().Context(), projectID, roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, blacklists)
}
