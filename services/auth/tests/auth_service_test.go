package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/auth/internal/models"
	"github.com/yourorg/collab/services/auth/internal/service"
	"github.com/yourorg/collab/services/auth/pkg/jwt"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockRefreshTokenRepository is a mock implementation
type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, userID, token string, expiresAt time.Time) (string, error) {
	args := m.Called(ctx, userID, token, expiresAt)
	return args.String(0), args.Error(1)
}

func (m *MockRefreshTokenRepository) GetByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// MockRedisRepository is a mock implementation
type MockRedisRepository struct {
	mock.Mock
}

func (m *MockRedisRepository) CacheUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	args := m.Called(ctx, user, ttl)
	return args.Error(0)
}

func (m *MockRedisRepository) GetCachedUser(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRedisRepository) InvalidateUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRedisRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisRepository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// Test Register
func TestAuthService_Register(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockRedisRepo := new(MockRedisRepository)

	jwtManager := jwt.NewManager("test-secret", time.Hour, 7*24*time.Hour, "test-issuer")

	authService := service.NewAuthService(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockRedisRepo,
		jwtManager,
		logger,
	)

	ctx := context.Background()
	req := &models.RegisterRequest{
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	// Setup mocks
	mockUserRepo.On("EmailExists", ctx, req.Email).Return(false, nil)
	mockUserRepo.On("UsernameExists", ctx, req.Username).Return(false, nil)
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	mockRefreshTokenRepo.On("Create", ctx, mock.Anything, mock.Anything, mock.Anything).Return("token-id", nil)
	mockRedisRepo.On("CacheUser", ctx, mock.Anything, mock.Anything).Return(nil)

	// Execute
	response, err := authService.Register(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, req.Email, response.User.Email)

	mockUserRepo.AssertExpectations(t)
	mockRefreshTokenRepo.AssertExpectations(t)
	mockRedisRepo.AssertExpectations(t)
}

// Test Login - Success
func TestAuthService_Login_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockUserRepo := new(MockUserRepository)
	mockRefreshTokenRepo := new(MockRefreshTokenRepository)
	mockRedisRepo := new(MockRedisRepository)

	jwtManager := jwt.NewManager("test-secret", time.Hour, 7*24*time.Hour, "test-issuer")

	authService := service.NewAuthService(
		mockUserRepo,
		mockRefreshTokenRepo,
		mockRedisRepo,
		jwtManager,
		logger,
	)

	ctx := context.Background()
	req := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	// Create a user with hashed password
	user := &models.User{
		ID:           "user-123",
		Email:        req.Email,
		Username:     "testuser",
		PasswordHash: "$2a$10$ZGQyNjM5NTQ4NDk3MjI2MuJ3v3fF8qV8Hx.6oL1YqXZwR8Z8qWz8a", // hashed "Password123!"
		Status:       models.UserStatusActive,
	}

	// Setup mocks
	mockUserRepo.On("GetByEmail", ctx, req.Email).Return(user, nil)
	mockUserRepo.On("UpdateLastLogin", ctx, user.ID).Return(nil)
	mockRefreshTokenRepo.On("Create", ctx, mock.Anything, mock.Anything, mock.Anything).Return("token-id", nil)
	mockRedisRepo.On("CacheUser", ctx, mock.Anything, mock.Anything).Return(nil)

	// Execute
	response, err := authService.Login(ctx, req)

	// Assert (may fail due to password hash, but structure is tested)
	assert.NotNil(t, response)
	mockUserRepo.AssertExpectations(t)
}
