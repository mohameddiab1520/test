package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourorg/collab/services/presence/internal/models"
	"github.com/yourorg/collab/services/presence/internal/repository"
	"go.uber.org/zap"
)

type PresenceService struct {
	presenceRepo *repository.PresenceRepository
	activityRepo *repository.ActivityRepository
	redis        *redis.Client
	logger       *zap.Logger
}

func NewPresenceService(
	presenceRepo *repository.PresenceRepository,
	activityRepo *repository.ActivityRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *PresenceService {
	return &PresenceService{
		presenceRepo: presenceRepo,
		activityRepo: activityRepo,
		redis:        redis,
		logger:       logger,
	}
}

func (s *PresenceService) GetSessionPresence(ctx context.Context, sessionID string) ([]*models.PresenceState, error) {
	return s.presenceRepo.GetSessionPresence(ctx, sessionID)
}

func (s *PresenceService) GetUserPresence(ctx context.Context, userID string) (*models.PresenceState, error) {
	return s.presenceRepo.GetUserPresence(ctx, userID)
}

func (s *PresenceService) UpdatePresence(ctx context.Context, req *models.UpdateStatusRequest) error {
	presence := &models.PresenceState{
		SessionID:       req.SessionID,
		UserID:          req.UserID,
		Status:          req.Status,
		CurrentScene:    req.CurrentScene,
		SelectedObject:  req.SelectedObject,
		CursorPosition:  req.CursorPosition,
		CameraTransform: req.CameraTransform,
		IsTyping:        req.IsTyping,
		LastActivityAt:  time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save to database
	if err := s.presenceRepo.UpsertPresence(ctx, presence); err != nil {
		s.logger.Error("Failed to upsert presence", zap.Error(err))
		return err
	}

	// Broadcast update via Redis pub/sub
	if err := s.presenceRepo.PublishPresenceUpdate(ctx, req.SessionID, presence); err != nil {
		s.logger.Warn("Failed to publish presence update", zap.Error(err))
	}

	// Create activity if object selected
	if req.SelectedObject != "" {
		activity := &models.ActivityFeedItem{
			SessionID:    req.SessionID,
			UserID:       req.UserID,
			ActivityType: models.ActivitySelected,
			ObjectID:     req.SelectedObject,
		}
		s.activityRepo.CreateActivity(ctx, activity)
	}

	return nil
}

func (s *PresenceService) ClearPresence(ctx context.Context, sessionID, userID string) error {
	// Mark as offline
	if err := s.presenceRepo.DeletePresence(ctx, sessionID, userID); err != nil {
		return err
	}

	// Create "left" activity
	activity := &models.ActivityFeedItem{
		SessionID:    sessionID,
		UserID:       userID,
		ActivityType: models.ActivityLeft,
	}
	s.activityRepo.CreateActivity(ctx, activity)

	return nil
}

func (s *PresenceService) GetActivityFeed(ctx context.Context, sessionID string, limit int) ([]*models.ActivityFeedItem, error) {
	if limit == 0 {
		limit = 50
	}
	return s.activityRepo.GetSessionActivities(ctx, sessionID, limit)
}

func (s *PresenceService) RecordActivity(ctx context.Context, activity *models.ActivityFeedItem) error {
	return s.activityRepo.CreateActivity(ctx, activity)
}
