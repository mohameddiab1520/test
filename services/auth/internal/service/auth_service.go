package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/yourorg/collab/services/auth/internal/models"
	"github.com/yourorg/collab/services/auth/internal/repository"
	"github.com/yourorg/collab/services/auth/pkg/jwt"
	"github.com/yourorg/collab/services/auth/pkg/password"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	redisRepo        *repository.RedisRepository
	jwtManager       *jwt.Manager
	logger           *zap.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	redisRepo *repository.RedisRepository,
	jwtManager *jwt.Manager,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		redisRepo:        redisRepo,
		jwtManager:       jwtManager,
		logger:           logger,
	}
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Validate password strength
	if err := password.Validate(req.Password); err != nil {
		return nil, err
	}

	// Check if email exists
	emailExists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check email existence", zap.Error(err))
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if emailExists {
		return nil, repository.ErrUserAlreadyExists
	}

	// Check if username exists
	usernameExists, err := s.userRepo.UsernameExists(ctx, req.Username)
	if err != nil {
		s.logger.Error("Failed to check username existence", zap.Error(err))
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if usernameExists {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	passwordHash, err := password.Hash(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		AvatarURL:    "", // Default empty
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User registered successfully", zap.String("userId", user.ID))

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	_, err = s.refreshTokenRepo.Create(ctx, user.ID, tokenPair.RefreshToken, expiresAt)
	if err != nil {
		s.logger.Error("Failed to store refresh token", zap.Error(err))
		// Don't fail the registration, just log
	}

	// Cache user data
	if err := s.redisRepo.CacheUser(ctx, user, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user", zap.Error(err))
		// Don't fail, just log
	}

	return &models.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         user.ToUserInfo(),
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, repository.ErrInvalidCredentials
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if user.Status != models.UserStatusActive {
		return nil, fmt.Errorf("user account is not active")
	}

	// Verify password
	if !password.Verify(req.Password, user.PasswordHash) {
		s.logger.Warn("Invalid login attempt", zap.String("email", req.Email))
		return nil, repository.ErrInvalidCredentials
	}

	// Update last login time
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login", zap.Error(err))
		// Don't fail the login
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = s.refreshTokenRepo.Create(ctx, user.ID, tokenPair.RefreshToken, expiresAt)
	if err != nil {
		s.logger.Error("Failed to store refresh token", zap.Error(err))
	}

	// Cache user data
	if err := s.redisRepo.CacheUser(ctx, user, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user", zap.Error(err))
	}

	s.logger.Info("User logged in successfully", zap.String("userId", user.ID))

	return &models.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         user.ToUserInfo(),
	}, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*models.AuthResponse, error) {
	// Validate refresh token
	userID, err := s.jwtManager.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if token exists and is not revoked
	storedToken, err := s.refreshTokenRepo.GetByToken(ctx, refreshTokenString)
	if err != nil {
		if err == repository.ErrRefreshTokenNotFound {
			return nil, fmt.Errorf("refresh token not found or revoked")
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Revoke old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, refreshTokenString); err != nil {
		s.logger.Warn("Failed to revoke old refresh token", zap.Error(err))
	}

	// Store new refresh token
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = s.refreshTokenRepo.Create(ctx, user.ID, tokenPair.RefreshToken, expiresAt)
	if err != nil {
		s.logger.Error("Failed to store new refresh token", zap.Error(err))
	}

	s.logger.Info("Token refreshed successfully", zap.String("userId", user.ID))

	return &models.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         user.ToUserInfo(),
	}, nil
}

// Logout logs out a user
func (s *AuthService) Logout(ctx context.Context, userID, refreshToken string) error {
	// Revoke refresh token
	if refreshToken != "" {
		if err := s.refreshTokenRepo.Revoke(ctx, refreshToken); err != nil {
			s.logger.Warn("Failed to revoke refresh token", zap.Error(err))
		}
	}

	// Delete all user sessions from Redis
	if err := s.redisRepo.DeleteAllUserSessions(ctx, userID); err != nil {
		s.logger.Warn("Failed to delete user sessions", zap.Error(err))
	}

	// Invalidate user cache
	if err := s.redisRepo.InvalidateUser(ctx, userID); err != nil {
		s.logger.Warn("Failed to invalidate user cache", zap.Error(err))
	}

	s.logger.Info("User logged out successfully", zap.String("userId", userID))

	return nil
}

// ValidateToken validates an access token
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	// Check if token is blacklisted
	blacklisted, err := s.redisRepo.IsTokenBlacklisted(ctx, token)
	if err != nil {
		s.logger.Warn("Failed to check token blacklist", zap.Error(err))
	}
	if blacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Validate token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Try to get user from cache first
	user, err := s.redisRepo.GetCachedUser(ctx, claims.UserID)
	if err != nil {
		s.logger.Warn("Failed to get cached user", zap.Error(err))
	}

	if user != nil {
		return user, nil
	}

	// Get user from database
	user, err = s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Cache user for future requests
	if err := s.redisRepo.CacheUser(ctx, user, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user", zap.Error(err))
	}

	return user, nil
}

// GetUserByID gets a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	// Try cache first
	user, err := s.redisRepo.GetCachedUser(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get cached user", zap.Error(err))
	}

	if user != nil {
		return user, nil
	}

	// Get from database
	user, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache user
	if err := s.redisRepo.CacheUser(ctx, user, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user", zap.Error(err))
	}

	return user, nil
}
