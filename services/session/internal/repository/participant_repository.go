package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourorg/collab/services/session/internal/models"
)

// ParticipantRepository handles participant data operations
type ParticipantRepository struct {
	db *sql.DB
}

// NewParticipantRepository creates a new participant repository
func NewParticipantRepository(db *sql.DB) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

// Join adds a participant to a session
func (r *ParticipantRepository) Join(ctx context.Context, sessionID, userID string, role models.ParticipantRole) error {
	query := `
		INSERT INTO session_participants (
			session_id, user_id, role, status, joined_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $5)
	`

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx, query,
		sessionID, userID, role, models.ParticipantStatusOnline, now,
	)

	if err != nil {
		// Check if it's a duplicate key error
		if err.Error() == "pq: duplicate key value violates unique constraint \"session_participants_pkey\"" {
			return ErrParticipantAlreadyJoined
		}
		return fmt.Errorf("failed to join session: %w", err)
	}

	return nil
}

// Leave removes a participant from a session
func (r *ParticipantRepository) Leave(ctx context.Context, sessionID, userID string) error {
	query := `
		UPDATE session_participants
		SET left_at = $3, status = 'offline'
		WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to leave session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrParticipantNotInSession
	}

	return nil
}

// GetBySessionAndUser retrieves a participant by session and user ID
func (r *ParticipantRepository) GetBySessionAndUser(ctx context.Context, sessionID, userID string) (*models.Participant, error) {
	query := `
		SELECT
			sp.session_id, sp.user_id, sp.role, sp.status,
			sp.current_scene, sp.selected_object, sp.cursor_position, sp.camera_transform,
			sp.voice_muted, sp.voice_deafened,
			sp.joined_at, sp.left_at, sp.last_activity_at,
			u.username, u.avatar_url
		FROM session_participants sp
		JOIN users u ON sp.user_id = u.user_id
		WHERE sp.session_id = $1 AND sp.user_id = $2
	`

	var participant models.Participant
	err := r.db.QueryRowContext(ctx, query, sessionID, userID).Scan(
		&participant.SessionID, &participant.UserID, &participant.Role, &participant.Status,
		&participant.CurrentScene, &participant.SelectedObject, &participant.CursorPosition,
		&participant.CameraTransform, &participant.VoiceMuted, &participant.VoiceDeafened,
		&participant.JoinedAt, &participant.LeftAt, &participant.LastActivityAt,
		&participant.Username, &participant.AvatarURL,
	)

	if err == sql.ErrNoRows {
		return nil, ErrParticipantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}

	return &participant, nil
}

// ListBySession retrieves all active participants in a session
func (r *ParticipantRepository) ListBySession(ctx context.Context, sessionID string) ([]models.Participant, error) {
	query := `
		SELECT
			sp.session_id, sp.user_id, sp.role, sp.status,
			sp.current_scene, sp.selected_object, sp.cursor_position, sp.camera_transform,
			sp.voice_muted, sp.voice_deafened,
			sp.joined_at, sp.left_at, sp.last_activity_at,
			u.username, u.avatar_url
		FROM session_participants sp
		JOIN users u ON sp.user_id = u.user_id
		WHERE sp.session_id = $1
		ORDER BY sp.joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	defer rows.Close()

	participants := []models.Participant{}
	for rows.Next() {
		var participant models.Participant
		err := rows.Scan(
			&participant.SessionID, &participant.UserID, &participant.Role, &participant.Status,
			&participant.CurrentScene, &participant.SelectedObject, &participant.CursorPosition,
			&participant.CameraTransform, &participant.VoiceMuted, &participant.VoiceDeafened,
			&participant.JoinedAt, &participant.LeftAt, &participant.LastActivityAt,
			&participant.Username, &participant.AvatarURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return participants, nil
}

// ListActiveBySession retrieves only active (not left) participants
func (r *ParticipantRepository) ListActiveBySession(ctx context.Context, sessionID string) ([]models.Participant, error) {
	query := `
		SELECT
			sp.session_id, sp.user_id, sp.role, sp.status,
			sp.current_scene, sp.selected_object, sp.cursor_position, sp.camera_transform,
			sp.voice_muted, sp.voice_deafened,
			sp.joined_at, sp.left_at, sp.last_activity_at,
			u.username, u.avatar_url
		FROM session_participants sp
		JOIN users u ON sp.user_id = u.user_id
		WHERE sp.session_id = $1 AND sp.left_at IS NULL
		ORDER BY sp.joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active participants: %w", err)
	}
	defer rows.Close()

	participants := []models.Participant{}
	for rows.Next() {
		var participant models.Participant
		err := rows.Scan(
			&participant.SessionID, &participant.UserID, &participant.Role, &participant.Status,
			&participant.CurrentScene, &participant.SelectedObject, &participant.CursorPosition,
			&participant.CameraTransform, &participant.VoiceMuted, &participant.VoiceDeafened,
			&participant.JoinedAt, &participant.LeftAt, &participant.LastActivityAt,
			&participant.Username, &participant.AvatarURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, participant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return participants, nil
}

// UpdatePresence updates participant presence information
func (r *ParticipantRepository) UpdatePresence(ctx context.Context, sessionID, userID string, req *models.UpdatePresenceRequest) error {
	query := `
		UPDATE session_participants
		SET
			status = COALESCE($3, status),
			current_scene = COALESCE($4, current_scene),
			selected_object = COALESCE($5, selected_object),
			cursor_position = COALESCE($6, cursor_position),
			camera_transform = COALESCE($7, camera_transform),
			voice_muted = COALESCE($8, voice_muted),
			voice_deafened = COALESCE($9, voice_deafened),
			last_activity_at = $10
		WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx, query,
		sessionID, userID,
		req.Status, req.CurrentScene, req.SelectedObject,
		req.CursorPosition, req.CameraTransform,
		req.VoiceMuted, req.VoiceDeafened,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update presence: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrParticipantNotInSession
	}

	return nil
}

// UpdateRole updates participant role
func (r *ParticipantRepository) UpdateRole(ctx context.Context, sessionID, userID string, role models.ParticipantRole) error {
	query := `
		UPDATE session_participants
		SET role = $3, last_activity_at = $4
		WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, userID, role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrParticipantNotInSession
	}

	return nil
}

// UpdateLastActivity updates the last activity timestamp
func (r *ParticipantRepository) UpdateLastActivity(ctx context.Context, sessionID, userID string) error {
	query := `
		UPDATE session_participants
		SET last_activity_at = $3
		WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last activity: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrParticipantNotInSession
	}

	return nil
}

// IsParticipant checks if a user is a participant in a session
func (r *ParticipantRepository) IsParticipant(ctx context.Context, sessionID, userID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM session_participants
			WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, sessionID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check participant: %w", err)
	}

	return exists, nil
}

// GetActiveCount gets the count of active participants
func (r *ParticipantRepository) GetActiveCount(ctx context.Context, sessionID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM session_participants
		WHERE session_id = $1 AND left_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active count: %w", err)
	}

	return count, nil
}
