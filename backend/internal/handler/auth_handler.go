package handler

import (
	"net/http"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	authRepo  domain.AuthRepository
	jwtSecret string
}

func NewAuthHandler(authRepo domain.AuthRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		authRepo:  authRepo,
		jwtSecret: jwtSecret,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register registers a new user
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email, password, and name are required"})
	}

	// Validate password length
	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 6 characters long"})
	}

	// Check existing user
	existing, err := h.authRepo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to check existing user"})
	}
	if existing != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "user with this email already exists"})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
	}

	// Create user
	user, err := h.authRepo.CreateUser(c.Request().Context(), req.Email, string(hash), req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	// Generate JWT (starts with empty projects list, user creates project next)
	token, err := GenerateJWTToken(h.jwtSecret, user, []string{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate authentication token"})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"token": token,
		"user":  user,
	})
}

// Login authenticates a user
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password are required"})
	}

	user, err := h.authRepo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to authenticate"})
	}
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	// Fetch user projects
	projects, err := h.authRepo.ListProjectsByOwnerID(c.Request().Context(), user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch user projects"})
	}

	projectIDs := make([]string, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Generate JWT
	token, err := GenerateJWTToken(h.jwtSecret, user, projectIDs)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate authentication token"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token":    token,
		"user":     user,
		"projects": projects,
	})
}

// Me retrieves current user info and projects
func (h *AuthHandler) Me(c echo.Context) error {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	user, err := h.authRepo.GetUserByID(c.Request().Context(), userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	projects, err := h.authRepo.ListProjectsByOwnerID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list projects"})
	}

	projectIDs := make([]string, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	// Return updated JWT token with current project list
	token, _ := GenerateJWTToken(h.jwtSecret, user, projectIDs)

	return c.JSON(http.StatusOK, map[string]any{
		"token":    token,
		"user":     user,
		"projects": projects,
	})
}
