package db

import (
	"context"

	"social-network/internal/models"
)

// CreateGroup creates a new group
func (s *SQLiteStore) CreateGroup(ctx context.Context, group *models.Group) error {
	query := `
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query, group.ID, group.CreatorID, group.Title, group.Description)
	return err
}

// GetGroups retrieves all groups available to browse
func (s *SQLiteStore) GetGroups(ctx context.Context) ([]*models.Group, error) {
	query := `SELECT id, creator_id, title, description, created_at FROM groups ORDER BY created_at DESC`
	rows, err := s.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]*models.Group, 0)
	for rows.Next() {
		g := &models.Group{}
		if err := rows.Scan(&g.ID, &g.CreatorID, &g.Title, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// AddGroupMember adds a user to a group with a specific status
func (s *SQLiteStore) AddGroupMember(ctx context.Context, groupID, userID, status string) error {
	query := `
		INSERT INTO group_members (group_id, user_id, status, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query, groupID, userID, status)
	return err
}

// CreateEvent creates a new event within a group
func (s *SQLiteStore) CreateEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, group_id, creator_id, title, description, event_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := s.ExecContext(ctx, query,
		event.ID,
		event.GroupID,
		event.CreatorID,
		event.Title,
		event.Description,
		event.EventDate,
	)
	return err
}

// RespondToEvent records a user's RSVP to an event
func (s *SQLiteStore) RespondToEvent(ctx context.Context, eventID, userID, status string) error {
	query := `
		INSERT INTO event_rsvps (event_id, user_id, status, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(event_id, user_id) DO UPDATE SET status = excluded.status
	`
	_, err := s.ExecContext(ctx, query, eventID, userID, status)
	return err
}
