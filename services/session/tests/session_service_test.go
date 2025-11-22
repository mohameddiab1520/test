package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/session/internal/models"
	"github.com/yourorg/collab/services/session/internal/service"
)

// MockSessionRepository is a mock implementation
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, sessionID string, req *models.UpdateSessionRequest) error {
	args := m.Called(ctx, sessionID, req)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateSceneData(ctx context.Context, sessionID string, sceneData *models.SceneData) error {
	args := m.Called(ctx, sessionID, sceneData)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) List(ctx context.Context, filter *models.SessionFilter) ([]models.Session, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]models.Session), args.Int(1), args.Error(2)
}

func (m *MockSessionRepository) GetParticipantCount(ctx context.Context, sessionID string) (int, error) {
	args := m.Called(ctx, sessionID)
	return args.Int(0), args.Error(1)
}

func (m *MockSessionRepository) StartSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) EndSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// MockParticipantRepository is a mock implementation
type MockParticipantRepository struct {
	mock.Mock
}

func (m *MockParticipantRepository) Join(ctx context.Context, sessionID, userID string, role models.ParticipantRole) error {
	args := m.Called(ctx, sessionID, userID, role)
	return args.Error(0)
}

func (m *MockParticipantRepository) Leave(ctx context.Context, sessionID, userID string) error {
	args := m.Called(ctx, sessionID, userID)
	return args.Error(0)
}

func (m *MockParticipantRepository) GetActiveCount(ctx context.Context, sessionID string) (int, error) {
	args := m.Called(ctx, sessionID)
	return args.Int(0), args.Error(1)
}

func (m *MockParticipantRepository) IsParticipant(ctx context.Context, sessionID, userID string) (bool, error) {
	args := m.Called(ctx, sessionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockParticipantRepository) ListActiveBySession(ctx context.Context, sessionID string) ([]models.Participant, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]models.Participant), args.Error(1)
}

func (m *MockParticipantRepository) UpdatePresence(ctx context.Context, sessionID, userID string, req *models.UpdatePresenceRequest) error {
	args := m.Called(ctx, sessionID, userID, req)
	return args.Error(0)
}

// MockRedisRepository is a mock implementation
type MockRedisRepository struct {
	mock.Mock
}

func (m *MockRedisRepository) CacheSession(ctx context.Context, session *models.Session, ttl interface{}) error {
	args := m.Called(ctx, session, ttl)
	return args.Error(0)
}

func (m *MockRedisRepository) GetCachedSession(ctx context.Context, sessionID string) (*models.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockRedisRepository) InvalidateSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockRedisRepository) PublishSessionEvent(ctx context.Context, sessionID string, event interface{}) error {
	args := m.Called(ctx, sessionID, event)
	return args.Error(0)
}

func (m *MockRedisRepository) UntrackConnection(ctx context.Context, sessionID, userID string) error {
	args := m.Called(ctx, sessionID, userID)
	return args.Error(0)
}

func (m *MockRedisRepository) UpdatePresenceCache(ctx context.Context, sessionID, userID string, presence map[string]interface{}, ttl interface{}) error {
	args := m.Called(ctx, sessionID, userID, presence, ttl)
	return args.Error(0)
}

// Test CreateSession
func TestSessionService_CreateSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockSessionRepo := new(MockSessionRepository)
	mockParticipantRepo := new(MockParticipantRepository)
	mockRedisRepo := new(MockRedisRepository)

	sessionService := service.NewSessionService(
		mockSessionRepo,
		mockParticipantRepo,
		mockRedisRepo,
		logger,
	)

	ctx := context.Background()
	userID := "user-123"
	req := &models.CreateSessionRequest{
		ProjectID:       "project-123",
		Name:            "Test Session",
		Description:     "Test description",
		MaxParticipants: 10,
		IsPublic:        true,
		VoiceEnabled:    true,
	}

	// Setup mocks
	mockSessionRepo.On("Create", ctx, mock.AnythingOfType("*models.Session")).Return(nil)
	mockParticipantRepo.On("Join", ctx, mock.Anything, userID, models.ParticipantRoleOwner).Return(nil)
	mockRedisRepo.On("CacheSession", ctx, mock.Anything, mock.Anything).Return(nil)

	// Execute
	session, err := sessionService.CreateSession(ctx, userID, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, req.Name, session.Name)
	assert.Equal(t, req.ProjectID, session.ProjectID)
	assert.Equal(t, userID, session.OwnerID)

	mockSessionRepo.AssertExpectations(t)
	mockParticipantRepo.AssertExpectations(t)
	mockRedisRepo.AssertExpectations(t)
}

// Test JoinSession
func TestSessionService_JoinSession(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockSessionRepo := new(MockSessionRepository)
	mockParticipantRepo := new(MockParticipantRepository)
	mockRedisRepo := new(MockRedisRepository)

	sessionService := service.NewSessionService(
		mockSessionRepo,
		mockParticipantRepo,
		mockRedisRepo,
		logger,
	)

	ctx := context.Background()
	sessionID := "session-123"
	userID := "user-456"

	session := &models.Session{
		ID:              sessionID,
		MaxParticipants: 10,
		Status:          models.SessionStatusActive,
	}

	// Setup mocks
	mockRedisRepo.On("GetCachedSession", ctx, sessionID).Return(session, nil)
	mockParticipantRepo.On("GetActiveCount", ctx, sessionID).Return(5, nil)
	mockParticipantRepo.On("Join", ctx, sessionID, userID, models.ParticipantRoleViewer).Return(nil)
	mockRedisRepo.On("PublishSessionEvent", ctx, sessionID, mock.Anything).Return(nil)

	// Execute
	err := sessionService.JoinSession(ctx, sessionID, userID, models.ParticipantRoleViewer)

	// Assert
	assert.NoError(t, err)

	mockSessionRepo.AssertExpectations(t)
	mockParticipantRepo.AssertExpectations(t)
	mockRedisRepo.AssertExpectations(t)
}
