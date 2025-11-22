package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/yourorg/collab/services/auth/internal/models"
	"github.com/yourorg/collab/services/auth/internal/repository"
	"github.com/yourorg/collab/services/auth/internal/service"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
	logger      *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration request"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	authResp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if err == repository.ErrUserAlreadyExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "User already exists",
				Message: "Email already registered",
			})
			return
		}

		h.logger.Error("Registration failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Registration failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, authResp)
}

// Login handles user login
// @Summary Login a user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	authResp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if err == repository.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid credentials",
				Message: "Email or password is incorrect",
			})
			return
		}

		h.logger.Error("Login failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Login failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	authResp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Token refresh failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, authResp)
}

// Logout handles user logout
// @Summary Logout a user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	// Get refresh token from request body (optional)
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	c.ShouldBindJSON(&req)

	err := h.authService.Logout(c.Request.Context(), userID.(string), req.RefreshToken)
	if err != nil {
		h.logger.Error("Logout failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Logout failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Logged out successfully",
	})
}

// GetCurrentUser gets the currently authenticated user
// @Summary Get current user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.UserInfo
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get user",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user.ToUserInfo())
}

// ValidateToken validates a token
// @Summary Validate access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body TokenValidationRequest true "Token to validate"
// @Success 200 {object} TokenValidationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/validate [post]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var req TokenValidationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		// Try to get from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid request",
				Message: "Token is required",
			})
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid authorization header",
				Message: "Format should be: Bearer <token>",
			})
			return
		}

		req.Token = parts[1]
	}

	user, err := h.authService.ValidateToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid token",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TokenValidationResponse{
		Valid:  true,
		UserID: user.ID,
		User:   user.ToUserInfo(),
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// TokenValidationRequest represents a token validation request
type TokenValidationRequest struct {
	Token string `json:"token" binding:"required"`
}

// TokenValidationResponse represents a token validation response
type TokenValidationResponse struct {
	Valid  bool             `json:"valid"`
	UserID string           `json:"userId"`
	User   *models.UserInfo `json:"user"`
}
