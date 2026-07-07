package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
)

type GroupRepository interface {
	CreateGroup(group *models.Group) error
	GetGroups() ([]*models.Group, error)
	GetGroupByID(id string) (*models.Group, error)
	CreateGroupMember(groupID, userID, status string) error
	GetGroupMembers(groupID string) ([]*models.User, error)
	GetGroupMembersByStatus(groupID, status string) ([]*models.User, error)
	GetGroupMemberStatus(groupID, userID string) (string, error)
	CreateEvent(event *models.Event) error
	GetGroupEvents(groupID string) ([]*models.Event, error)
	CreateEventRSVP(eventID, userID, status string) error
}

type sqliteGroupRepo struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) GroupRepository {
	return &sqliteGroupRepo{db: db}
}

func (r *sqliteGroupRepo) CreateGroup(group *models.Group) error {
	const query = `
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, group.ID, group.CreatorID, group.Title, group.Description, group.CreatedAt)
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	// Creator is automatically an accepted member
	return r.CreateGroupMember(group.ID, group.CreatorID, "accepted")
}

func (r *sqliteGroupRepo) GetGroups() ([]*models.Group, error) {
	const query = `SELECT id, creator_id, title, description, created_at FROM groups ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		g := &models.Group{}
		if err := rows.Scan(&g.ID, &g.CreatorID, &g.Title, &g.Description, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *sqliteGroupRepo) GetGroupByID(id string) (*models.Group, error) {
	const query = `SELECT id, creator_id, title, description, created_at FROM groups WHERE id = ?`
	g := &models.Group{}
	err := r.db.QueryRow(query, id).Scan(&g.ID, &g.CreatorID, &g.Title, &g.Description, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("group not found")
		}
		return nil, fmt.Errorf("get group by id: %w", err)
	}
	return g, nil
}

func (r *sqliteGroupRepo) CreateGroupMember(groupID, userID, status string) error {
	const query = `
		INSERT INTO group_members (group_id, user_id, status, created_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(group_id, user_id) DO UPDATE SET status = excluded.status`
	_, err := r.db.Exec(query, groupID, userID, status)
	if err != nil {
		return fmt.Errorf("create/update group member: %w", err)
	}
	return nil
}

func (r *sqliteGroupRepo) GetGroupMembers(groupID string) ([]*models.User, error) {
	return r.getGroupMembersByStatus(groupID, "accepted")
}

func (r *sqliteGroupRepo) GetGroupMembersByStatus(groupID, status string) ([]*models.User, error) {
	return r.getGroupMembersByStatus(groupID, status)
}

func (r *sqliteGroupRepo) getGroupMembersByStatus(groupID, status string) ([]*models.User, error) {
	const query = `
		SELECT u.id, u.email, u.first_name, u.last_name, u.dob,
		       COALESCE(u.nickname, ''), COALESCE(u.about_me, ''), COALESCE(u.avatar_path, ''),
		       u.is_public, u.created_at
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = ? AND gm.status = ?
		ORDER BY gm.created_at DESC`

	rows, err := r.db.Query(query, groupID, status)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.DOB,
			&u.Nickname, &u.AboutMe, &u.AvatarPath, &u.IsPublic, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan group member: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *sqliteGroupRepo) GetGroupMemberStatus(groupID, userID string) (string, error) {
	const query = `SELECT status FROM group_members WHERE group_id = ? AND user_id = ?`
	var status string
	err := r.db.QueryRow(query, groupID, userID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "none", nil
		}
		return "", fmt.Errorf("get group member status: %w", err)
	}
	return status, nil
}

func (r *sqliteGroupRepo) CreateEvent(event *models.Event) error {
	const query = `
		INSERT INTO events (id, group_id, creator_id, title, description, event_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, event.ID, event.GroupID, event.CreatorID, event.Title, event.Description, event.EventDate, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (r *sqliteGroupRepo) GetGroupEvents(groupID string) ([]*models.Event, error) {
	const query = `SELECT id, group_id, creator_id, title, description, event_date, created_at FROM events WHERE group_id = ? ORDER BY event_date ASC`
	rows, err := r.db.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("get group events: %w", err)
	}
	defer rows.Close()

	var events []*models.Event
	for rows.Next() {
		e := &models.Event{}
		if err := rows.Scan(&e.ID, &e.GroupID, &e.CreatorID, &e.Title, &e.Description, &e.EventDate, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *sqliteGroupRepo) CreateEventRSVP(eventID, userID, status string) error {
	const query = `
		INSERT INTO event_rsvps (event_id, user_id, status, created_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(event_id, user_id) DO UPDATE SET status = excluded.status`
	_, err := r.db.Exec(query, eventID, userID, status)
	if err != nil {
		return fmt.Errorf("create/update event rsvp: %w", err)
	}
	return nil
}
