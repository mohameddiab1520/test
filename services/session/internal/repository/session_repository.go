package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourorg/collab/services/session/internal/models"
)

// SessionRepository handles session data operations
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (
			session_id, project_id, owner_id, name, description,
			settings, max_participants, is_public, voice_enabled, record_session,
			status, scene_data, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING session_id, created_at, updated_at
	`

	now := time.Now()
	err := r.db.QueryRowContext(
		ctx, query,
		session.ID, session.ProjectID, session.OwnerID, session.Name, session.Description,
		session.Settings, session.MaxParticipants, session.IsPublic, session.VoiceEnabled,
		session.RecordSession, session.Status, session.SceneData, now, now,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	query := `
		SELECT
			session_id, project_id, owner_id, name, description,
			settings, max_participants, is_public, voice_enabled, record_session,
			status, scene_data, recording_url,
			created_at, updated_at, started_at, ended_at
		FROM sessions
		WHERE session_id = $1
	`

	var session models.Session
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.ProjectID, &session.OwnerID, &session.Name, &session.Description,
		&session.Settings, &session.MaxParticipants, &session.IsPublic, &session.VoiceEnabled,
		&session.RecordSession, &session.Status, &session.SceneData, &session.RecordingURL,
		&session.CreatedAt, &session.UpdatedAt, &session.StartedAt, &session.EndedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, sessionID string, req *models.UpdateSessionRequest) error {
	query := `
		UPDATE sessions
		SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			settings = COALESCE($4, settings),
			max_participants = COALESCE($5, max_participants),
			is_public = COALESCE($6, is_public),
			voice_enabled = COALESCE($7, voice_enabled),
			status = COALESCE($8, status),
			updated_at = $9
		WHERE session_id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		sessionID, req.Name, req.Description, req.Settings, req.MaxParticipants,
		req.IsPublic, req.VoiceEnabled, req.Status, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateSceneData updates session scene data
func (r *SessionRepository) UpdateSceneData(ctx context.Context, sessionID string, sceneData *models.SceneData) error {
	query := `
		UPDATE sessions
		SET scene_data = $2, updated_at = $3
		WHERE session_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, sceneData, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update scene data: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// Delete deletes a session (soft delete by setting status to 'ended')
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	query := `
		UPDATE sessions
		SET status = 'ended', ended_at = $2, updated_at = $2
		WHERE session_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// List retrieves sessions with filters and pagination
func (r *SessionRepository) List(ctx context.Context, filter *models.SessionFilter) ([]models.Session, int, error) {
	// Build query dynamically based on filters
	query := `
		SELECT
			session_id, project_id, owner_id, name, description,
			settings, max_participants, is_public, voice_enabled, record_session,
			status, scene_data, recording_url,
			created_at, updated_at, started_at, ended_at
		FROM sessions
		WHERE 1=1
	`

	countQuery := "SELECT COUNT(*) FROM sessions WHERE 1=1"

	args := []interface{}{}
	argCount := 1

	// Add filters
	if filter.ProjectID != "" {
		query += fmt.Sprintf(" AND project_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND project_id = $%d", argCount)
		args = append(args, filter.ProjectID)
		argCount++
	}

	if filter.OwnerID != "" {
		query += fmt.Sprintf(" AND owner_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND owner_id = $%d", argCount)
		args = append(args, filter.OwnerID)
		argCount++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.IsPublic != nil {
		query += fmt.Sprintf(" AND is_public = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_public = $%d", argCount)
		args = append(args, *filter.IsPublic)
		argCount++
	}

	// Get total count
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if filter.PageSize > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
		args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []models.Session{}
	for rows.Next() {
		var session models.Session
		err := rows.Scan(
			&session.ID, &session.ProjectID, &session.OwnerID, &session.Name, &session.Description,
			&session.Settings, &session.MaxParticipants, &session.IsPublic, &session.VoiceEnabled,
			&session.RecordSession, &session.Status, &session.SceneData, &session.RecordingURL,
			&session.CreatedAt, &session.UpdatedAt, &session.StartedAt, &session.EndedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return sessions, totalCount, nil
}

// GetParticipantCount gets the current participant count for a session
func (r *SessionRepository) GetParticipantCount(ctx context.Context, sessionID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM session_participants
		WHERE session_id = $1 AND left_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get participant count: %w", err)
	}

	return count, nil
}

// StartSession marks a session as started
func (r *SessionRepository) StartSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE sessions
		SET status = 'active', started_at = $2, updated_at = $2
		WHERE session_id = $1 AND status != 'ended'
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// EndSession marks a session as ended
func (r *SessionRepository) EndSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE sessions
		SET status = 'ended', ended_at = $2, updated_at = $2
		WHERE session_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}
