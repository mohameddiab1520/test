package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/yourorg/collab/services/presence/internal/models"
)

type PresenceRepository struct {
	db    *sql.DB
	redis *redis.Client
}

func NewPresenceRepository(db *sql.DB, redis *redis.Client) *PresenceRepository {
	return &PresenceRepository{
		db:    db,
		redis: redis,
	}
}

func (r *PresenceRepository) GetSessionPresence(ctx context.Context, sessionID string) ([]*models.PresenceState, error) {
	query := `
		SELECT
			p.presence_id, p.session_id, p.user_id, p.status, p.current_scene,
			p.selected_object, p.cursor_position, p.camera_transform, p.is_typing,
			p.last_activity_at, p.updated_at, u.name, u.avatar_url
		FROM presence_states p
		JOIN users u ON p.user_id = u.user_id
		WHERE p.session_id = $1
		  AND p.status != 'offline'
		ORDER BY p.last_activity_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session presence: %w", err)
	}
	defer rows.Close()

	var presences []*models.PresenceState
	for rows.Next() {
		presence := &models.PresenceState{}
		var cursorJSON, cameraJSON []byte

		err := rows.Scan(
			&presence.PresenceID,
			&presence.SessionID,
			&presence.UserID,
			&presence.Status,
			&presence.CurrentScene,
			&presence.SelectedObject,
			&cursorJSON,
			&cameraJSON,
			&presence.IsTyping,
			&presence.LastActivityAt,
			&presence.UpdatedAt,
			&presence.UserName,
			&presence.UserAvatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan presence: %w", err)
		}

		if len(cursorJSON) > 0 {
			json.Unmarshal(cursorJSON, &presence.CursorPosition)
		}
		if len(cameraJSON) > 0 {
			json.Unmarshal(cameraJSON, &presence.CameraTransform)
		}

		presences = append(presences, presence)
	}

	return presences, nil
}

func (r *PresenceRepository) GetUserPresence(ctx context.Context, userID string) (*models.PresenceState, error) {
	query := `
		SELECT
			p.presence_id, p.session_id, p.user_id, p.status, p.current_scene,
			p.selected_object, p.cursor_position, p.camera_transform, p.is_typing,
			p.last_activity_at, p.updated_at, u.name, u.avatar_url
		FROM presence_states p
		JOIN users u ON p.user_id = u.user_id
		WHERE p.user_id = $1
		ORDER BY p.last_activity_at DESC
		LIMIT 1
	`

	presence := &models.PresenceState{}
	var cursorJSON, cameraJSON []byte

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&presence.PresenceID,
		&presence.SessionID,
		&presence.UserID,
		&presence.Status,
		&presence.CurrentScene,
		&presence.SelectedObject,
		&cursorJSON,
		&cameraJSON,
		&presence.IsTyping,
		&presence.LastActivityAt,
		&presence.UpdatedAt,
		&presence.UserName,
		&presence.UserAvatar,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user presence: %w", err)
	}

	if len(cursorJSON) > 0 {
		json.Unmarshal(cursorJSON, &presence.CursorPosition)
	}
	if len(cameraJSON) > 0 {
		json.Unmarshal(cameraJSON, &presence.CameraTransform)
	}

	return presence, nil
}

func (r *PresenceRepository) UpsertPresence(ctx context.Context, presence *models.PresenceState) error {
	if presence.PresenceID == "" {
		presence.PresenceID = uuid.New().String()
	}

	cursorJSON, _ := json.Marshal(presence.CursorPosition)
	cameraJSON, _ := json.Marshal(presence.CameraTransform)

	query := `
		INSERT INTO presence_states (
			presence_id, session_id, user_id, status, current_scene,
			selected_object, cursor_position, camera_transform, is_typing,
			last_activity_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (session_id, user_id)
		DO UPDATE SET
			status = $4,
			current_scene = $5,
			selected_object = $6,
			cursor_position = $7,
			camera_transform = $8,
			is_typing = $9,
			last_activity_at = $10,
			updated_at = $11
	`

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx, query,
		presence.PresenceID,
		presence.SessionID,
		presence.UserID,
		presence.Status,
		presence.CurrentScene,
		presence.SelectedObject,
		cursorJSON,
		cameraJSON,
		presence.IsTyping,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert presence: %w", err)
	}

	// Cache in Redis
	key := fmt.Sprintf("presence:%s:%s", presence.SessionID, presence.UserID)
	data, _ := json.Marshal(presence)
	r.redis.Set(ctx, key, data, 5*time.Minute)

	return nil
}

func (r *PresenceRepository) DeletePresence(ctx context.Context, sessionID, userID string) error {
	query := `
		UPDATE presence_states
		SET status = 'offline', updated_at = $3
		WHERE session_id = $1 AND user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, sessionID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete presence: %w", err)
	}

	// Remove from Redis
	key := fmt.Sprintf("presence:%s:%s", sessionID, userID)
	r.redis.Del(ctx, key)

	return nil
}

func (r *PresenceRepository) PublishPresenceUpdate(ctx context.Context, sessionID string, presence *models.PresenceState) error {
	channel := fmt.Sprintf("presence:%s", sessionID)
	data, _ := json.Marshal(presence)
	return r.redis.Publish(ctx, channel, data).Err()
}
