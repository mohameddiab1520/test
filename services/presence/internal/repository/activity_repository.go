package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/collab/services/presence/internal/models"
)

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) CreateActivity(ctx context.Context, activity *models.ActivityFeedItem) error {
	activity.ActivityID = uuid.New().String()
	activity.CreatedAt = time.Now()

	metadataJSON, _ := json.Marshal(activity.Metadata)

	query := `
		INSERT INTO activity_feed (
			activity_id, session_id, user_id, activity_type,
			object_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		activity.ActivityID,
		activity.SessionID,
		activity.UserID,
		activity.ActivityType,
		activity.ObjectID,
		metadataJSON,
		activity.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}

	return nil
}

func (r *ActivityRepository) GetSessionActivities(ctx context.Context, sessionID string, limit int) ([]*models.ActivityFeedItem, error) {
	query := `
		SELECT
			a.activity_id, a.session_id, a.user_id, a.activity_type,
			a.object_id, a.metadata, a.created_at, u.name
		FROM activity_feed a
		JOIN users u ON a.user_id = u.user_id
		WHERE a.session_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer rows.Close()

	var activities []*models.ActivityFeedItem
	for rows.Next() {
		activity := &models.ActivityFeedItem{}
		var metadataJSON []byte

		err := rows.Scan(
			&activity.ActivityID,
			&activity.SessionID,
			&activity.UserID,
			&activity.ActivityType,
			&activity.ObjectID,
			&metadataJSON,
			&activity.CreatedAt,
			&activity.UserName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &activity.Metadata)
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

func (r *ActivityRepository) CleanOldActivities(ctx context.Context, daysToKeep int) error {
	query := `
		DELETE FROM activity_feed
		WHERE created_at < $1
	`

	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)
	_, err := r.db.ExecContext(ctx, query, cutoffDate)

	return err
}
