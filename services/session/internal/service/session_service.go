package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/yourorg/collab/services/session/internal/models"
	"github.com/yourorg/collab/services/session/internal/repository"
)

// SessionService handles session business logic
type SessionService struct {
	sessionRepo     *repository.SessionRepository
	participantRepo *repository.ParticipantRepository
	redisRepo       *repository.RedisRepository
	logger          *zap.Logger
}

// NewSessionService creates a new session service
func NewSessionService(
	sessionRepo *repository.SessionRepository,
	participantRepo *repository.ParticipantRepository,
	redisRepo *repository.RedisRepository,
	logger *zap.Logger,
) *SessionService {
	return &SessionService{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		redisRepo:       redisRepo,
		logger:          logger,
	}
}

// CreateSession creates a new collaboration session
func (s *SessionService) CreateSession(ctx context.Context, userID string, req *models.CreateSessionRequest) (*models.Session, error) {
	// Create session
	session := &models.Session{
		ID:              uuid.New().String(),
		ProjectID:       req.ProjectID,
		OwnerID:         userID,
		Name:            req.Name,
		Description:     req.Description,
		Settings:        req.Settings,
		MaxParticipants: req.MaxParticipants,
		IsPublic:        req.IsPublic,
		VoiceEnabled:    req.VoiceEnabled,
		RecordSession:   req.RecordSession,
		Status:          models.SessionStatusActive,
	}

	// Save to database
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		s.logger.Error("Failed to create session", zap.Error(err))
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Auto-join owner as participant
	if err := s.participantRepo.Join(ctx, session.ID, userID, models.ParticipantRoleOwner); err != nil {
		s.logger.Error("Failed to add owner as participant", zap.Error(err))
		// Don't fail the session creation, just log
	}

	// Cache session
	if err := s.redisRepo.CacheSession(ctx, session, 2*time.Hour); err != nil {
		s.logger.Warn("Failed to cache session", zap.Error(err))
	}

	s.logger.Info("Session created successfully",
		zap.String("sessionId", session.ID),
		zap.String("ownerId", userID),
	)

	return session, nil
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	// Try cache first
	session, err := s.redisRepo.GetCachedSession(ctx, sessionID)
	if err != nil {
		s.logger.Warn("Failed to get cached session", zap.Error(err))
	}

	if session != nil {
		return session, nil
	}

	// Get from database
	session, err = s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	if err := s.redisRepo.CacheSession(ctx, session, 2*time.Hour); err != nil {
		s.logger.Warn("Failed to cache session", zap.Error(err))
	}

	return session, nil
}

// UpdateSession updates a session
func (s *SessionService) UpdateSession(ctx context.Context, sessionID, userID string, req *models.UpdateSessionRequest) (*models.Session, error) {
	// Get session to check ownership
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Check if user is the owner
	if session.OwnerID != userID {
		return nil, repository.ErrNotSessionOwner
	}

	// Check if session has ended
	if session.Status == models.SessionStatusEnded {
		return nil, repository.ErrSessionEnded
	}

	// Update session
	if err := s.sessionRepo.Update(ctx, sessionID, req); err != nil {
		s.logger.Error("Failed to update session", zap.Error(err))
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Invalidate cache
	if err := s.redisRepo.InvalidateSession(ctx, sessionID); err != nil {
		s.logger.Warn("Failed to invalidate session cache", zap.Error(err))
	}

	// Get updated session
	session, err = s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Publish update event
	event := map[string]interface{}{
		"type":      "session_updated",
		"sessionId": sessionID,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish session update event", zap.Error(err))
	}

	s.logger.Info("Session updated successfully", zap.String("sessionId", sessionID))

	return session, nil
}

// UpdateSceneData updates session scene data
func (s *SessionService) UpdateSceneData(ctx context.Context, sessionID, userID string, sceneData *models.SceneData) error {
	// Check if user is a participant
	isParticipant, err := s.participantRepo.IsParticipant(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	if !isParticipant {
		return repository.ErrParticipantNotInSession
	}

	// Update scene data
	if err := s.sessionRepo.UpdateSceneData(ctx, sessionID, sceneData); err != nil {
		s.logger.Error("Failed to update scene data", zap.Error(err))
		return fmt.Errorf("failed to update scene data: %w", err)
	}

	// Invalidate cache
	if err := s.redisRepo.InvalidateSession(ctx, sessionID); err != nil {
		s.logger.Warn("Failed to invalidate session cache", zap.Error(err))
	}

	// Publish scene update event
	event := map[string]interface{}{
		"type":      "scene_updated",
		"sessionId": sessionID,
		"userId":    userID,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish scene update event", zap.Error(err))
	}

	s.logger.Info("Scene data updated successfully", zap.String("sessionId", sessionID))

	return nil
}

// DeleteSession deletes a session (soft delete)
func (s *SessionService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	// Get session to check ownership
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check if user is the owner
	if session.OwnerID != userID {
		return repository.ErrNotSessionOwner
	}

	// End session
	if err := s.sessionRepo.EndSession(ctx, sessionID); err != nil {
		s.logger.Error("Failed to delete session", zap.Error(err))
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Invalidate cache
	if err := s.redisRepo.InvalidateSession(ctx, sessionID); err != nil {
		s.logger.Warn("Failed to invalidate session cache", zap.Error(err))
	}

	// Publish session end event
	event := map[string]interface{}{
		"type":      "session_ended",
		"sessionId": sessionID,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish session end event", zap.Error(err))
	}

	s.logger.Info("Session deleted successfully", zap.String("sessionId", sessionID))

	return nil
}

// ListSessions lists sessions with filters and pagination
func (s *SessionService) ListSessions(ctx context.Context, filter *models.SessionFilter) (*models.SessionListResponse, error) {
	// Set default pagination
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}

	sessions, totalCount, err := s.sessionRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list sessions", zap.Error(err))
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	hasMore := (filter.Page * filter.PageSize) < totalCount

	return &models.SessionListResponse{
		Sessions:   sessions,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		HasMore:    hasMore,
	}, nil
}

// JoinSession adds a participant to a session
func (s *SessionService) JoinSession(ctx context.Context, sessionID, userID string, role models.ParticipantRole) error {
	// Get session
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check if session has ended
	if session.Status == models.SessionStatusEnded {
		return repository.ErrSessionEnded
	}

	// Check if session is full
	activeCount, err := s.participantRepo.GetActiveCount(ctx, sessionID)
	if err != nil {
		return err
	}

	if activeCount >= session.MaxParticipants {
		return repository.ErrSessionFull
	}

	// Join session
	if err := s.participantRepo.Join(ctx, sessionID, userID, role); err != nil {
		s.logger.Error("Failed to join session", zap.Error(err))
		return fmt.Errorf("failed to join session: %w", err)
	}

	// Publish join event
	event := map[string]interface{}{
		"type":      "participant_joined",
		"sessionId": sessionID,
		"userId":    userID,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish join event", zap.Error(err))
	}

	s.logger.Info("User joined session",
		zap.String("sessionId", sessionID),
		zap.String("userId", userID),
	)

	return nil
}

// LeaveSession removes a participant from a session
func (s *SessionService) LeaveSession(ctx context.Context, sessionID, userID string) error {
	// Leave session
	if err := s.participantRepo.Leave(ctx, sessionID, userID); err != nil {
		s.logger.Error("Failed to leave session", zap.Error(err))
		return fmt.Errorf("failed to leave session: %w", err)
	}

	// Untrack WebSocket connection
	if err := s.redisRepo.UntrackConnection(ctx, sessionID, userID); err != nil {
		s.logger.Warn("Failed to untrack connection", zap.Error(err))
	}

	// Publish leave event
	event := map[string]interface{}{
		"type":      "participant_left",
		"sessionId": sessionID,
		"userId":    userID,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish leave event", zap.Error(err))
	}

	s.logger.Info("User left session",
		zap.String("sessionId", sessionID),
		zap.String("userId", userID),
	)

	return nil
}

// GetParticipants retrieves all participants in a session
func (s *SessionService) GetParticipants(ctx context.Context, sessionID string) (*models.ParticipantListResponse, error) {
	participants, err := s.participantRepo.ListActiveBySession(ctx, sessionID)
	if err != nil {
		s.logger.Error("Failed to get participants", zap.Error(err))
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return &models.ParticipantListResponse{
		Participants: participants,
		TotalCount:   len(participants),
	}, nil
}

// UpdatePresence updates participant presence
func (s *SessionService) UpdatePresence(ctx context.Context, sessionID, userID string, req *models.UpdatePresenceRequest) error {
	// Update presence in database
	if err := s.participantRepo.UpdatePresence(ctx, sessionID, userID, req); err != nil {
		return err
	}

	// Update presence cache
	presence := map[string]interface{}{
		"status":      req.Status,
		"lastActivity": time.Now().Unix(),
	}

	if req.CurrentScene != nil {
		presence["currentScene"] = *req.CurrentScene
	}
	if req.SelectedObject != nil {
		presence["selectedObject"] = *req.SelectedObject
	}
	if req.CursorPosition != nil {
		presence["cursorPosition"] = req.CursorPosition
	}
	if req.CameraTransform != nil {
		presence["cameraTransform"] = req.CameraTransform
	}

	if err := s.redisRepo.UpdatePresenceCache(ctx, sessionID, userID, presence, 5*time.Minute); err != nil {
		s.logger.Warn("Failed to update presence cache", zap.Error(err))
	}

	// Publish presence update event
	event := map[string]interface{}{
		"type":      "presence_updated",
		"sessionId": sessionID,
		"userId":    userID,
		"presence":  presence,
		"timestamp": time.Now().Unix(),
	}
	if err := s.redisRepo.PublishSessionEvent(ctx, sessionID, event); err != nil {
		s.logger.Warn("Failed to publish presence update event", zap.Error(err))
	}

	return nil
}
