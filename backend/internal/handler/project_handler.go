package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
)

type ProjectHandler struct {
	authRepo  domain.AuthRepository
	d1Store   domain.D1Store
	jwtSecret string
}

func NewProjectHandler(authRepo domain.AuthRepository, d1Store domain.D1Store, jwtSecret string) *ProjectHandler {
	return &ProjectHandler{
		authRepo:  authRepo,
		d1Store:   d1Store,
		jwtSecret: jwtSecret,
	}
}

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateAPIKeyRequest struct {
	Name          string `json:"name"`
	ExpiresInDays *int   `json:"expires_in_days"`
}

// ListProjects lists projects accessible to the logged in user
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	projects, err := h.authRepo.ListProjectsByOwnerID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list projects"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"projects": projects,
	})
}

// CreateProject creates a new project for the logged in user
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project name is required"})
	}

	project, err := h.authRepo.CreateProject(c.Request().Context(), req.Name, req.Description, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
	}

	// Dynamic Provisioning of Cloudflare D1 Database (Asynchronous Queue)
	dbName := fmt.Sprintf("ck_proj_%s", strings.ReplaceAll(project.ID, "-", ""))
	if len(dbName) > 32 {
		dbName = dbName[:32]
	}

	err = h.authRepo.CreatePendingD1Infrastructure(c.Request().Context(), project.ID, dbName)
	if err != nil {
		fmt.Printf("[Project Handler Error]: failed to enqueue D1 provisioning for project %s: %v\n", project.ID, err)
	}

	// Generate updated JWT token for user containing the new project ID
	user, _ := h.authRepo.GetUserByID(c.Request().Context(), userID)
	allProjects, _ := h.authRepo.ListProjectsByOwnerID(c.Request().Context(), userID)
	pIDs := make([]string, len(allProjects))
	for i, p := range allProjects {
		pIDs[i] = p.ID
	}
	newToken, _ := GenerateJWTToken(h.jwtSecret, user, pIDs)

	return c.JSON(http.StatusCreated, map[string]any{
		"project": project,
		"token":   newToken,
	})
}

// DeleteProject soft-deletes a project owned by the user
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_id param required"})
	}

	err := h.authRepo.SoftDeleteProject(c.Request().Context(), projectID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// GetProjectD1Status returns the provisioning status of the Cloudflare D1 database for a project
func (h *ProjectHandler) GetProjectD1Status(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	projectID := c.Param("project_id")

	// Check user membership / ownership
	isAuth, err := h.authRepo.IsUserProjectMemberOrOwner(c.Request().Context(), projectID, userID)
	if err != nil || !isAuth {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied for this project"})
	}

	status, err := h.authRepo.GetD1Status(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve d1 status"})
	}

	if status == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "infrastructure record not found"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"d1_status": status,
	})
}

// ListAPIKeys lists API keys for a given project
func (h *ProjectHandler) ListAPIKeys(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	projectID := c.Param("project_id")

	// Check user membership / ownership
	isAuth, err := h.authRepo.IsUserProjectMemberOrOwner(c.Request().Context(), projectID, userID)
	if err != nil || !isAuth {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied for this project"})
	}

	keys, err := h.authRepo.ListAPIKeysByProjectID(c.Request().Context(), projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list API keys"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"api_keys": keys,
	})
}

// CreateAPIKey generates a new API key for a project
func (h *ProjectHandler) CreateAPIKey(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	projectID := c.Param("project_id")

	isAuth, err := h.authRepo.IsUserProjectMemberOrOwner(c.Request().Context(), projectID, userID)
	if err != nil || !isAuth {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied for this project"})
	}

	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = "API Key"
	}

	// Generate random 32-byte hex key
	rawBytes := make([]byte, 16)
	if _, err := rand.Read(rawBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate random key"})
	}
	randomHex := hex.EncodeToString(rawBytes)
	rawKey := fmt.Sprintf("ck_live_%s", randomHex)
	keyPrefix := fmt.Sprintf("ck_live_%s...", randomHex[:6])

	// Hash key
	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := fmt.Sprintf("%x", hashBytes)

	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		t := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		expiresAt = &t
	}

	apiKey, err := h.authRepo.CreateAPIKey(c.Request().Context(), projectID, req.Name, keyHash, keyPrefix, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"api_key": apiKey,
		"raw_key": rawKey, // Displayed ONLY ONCE
	})
}

// RevokeAPIKey revokes an API key
func (h *ProjectHandler) RevokeAPIKey(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	projectID := c.Param("project_id")
	keyID := c.Param("key_id")

	isAuth, err := h.authRepo.IsUserProjectMemberOrOwner(c.Request().Context(), projectID, userID)
	if err != nil || !isAuth {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied for this project"})
	}

	err = h.authRepo.RevokeAPIKey(c.Request().Context(), keyID, projectID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "revoked"})
}
