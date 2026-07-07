package repository

import "testing"

func TestGetGroupMembersScansBooleanVisibility(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGroupRepository(db)

	_, err := db.Exec(`
		INSERT INTO users (id, email, password, first_name, last_name, dob, nickname, about_me, is_public, created_at)
		VALUES
			('creator-1', 'creator@example.com', 'hash', 'Creator', 'One', '1990-01-01', 'creator', '', 1, datetime('now')),
			('member-1', 'member@example.com', 'hash', 'Member', 'One', '1991-01-01', 'member', '', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("seed users: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES ('group-1', 'creator-1', 'Group One', 'Description', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO group_members (group_id, user_id, status, created_at)
		VALUES
			('group-1', 'creator-1', 'accepted', datetime('now')),
			('group-1', 'member-1', 'accepted', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group members: %v", err)
	}

	members, err := repo.GetGroupMembers("group-1")
	if err != nil {
		t.Fatalf("get group members: %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	if !members[0].IsPublic && !members[1].IsPublic {
		t.Fatalf("expected scanned boolean visibility to be preserved")
	}
}

func TestCreateGroupMemberAllowsDeclinedStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGroupRepository(db)

	_, err := db.Exec(`
		INSERT INTO users (id, email, password, first_name, last_name, dob, nickname, about_me, is_public, created_at)
		VALUES
			('creator-1', 'creator@example.com', 'hash', 'Creator', 'One', '1990-01-01', 'creator', '', 1, datetime('now')),
			('invitee-1', 'invitee@example.com', 'hash', 'Invitee', 'One', '1991-01-01', 'invitee', '', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("seed users: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES ('group-1', 'creator-1', 'Group One', 'Description', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}

	if err := repo.CreateGroupMember("group-1", "invitee-1", "declined"); err != nil {
		t.Fatalf("create declined group member: %v", err)
	}

	status, err := repo.GetGroupMemberStatus("group-1", "invitee-1")
	if err != nil {
		t.Fatalf("get group member status: %v", err)
	}
	if status != "declined" {
		t.Fatalf("expected declined status, got %q", status)
	}
}

func TestGetEventAttendeesReturnsGoingUsers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGroupRepository(db)

	_, err := db.Exec(`
		INSERT INTO users (id, email, password, first_name, last_name, dob, nickname, about_me, is_public, created_at)
		VALUES
			('creator-1', 'creator@example.com', 'hash', 'Creator', 'One', '1990-01-01', 'creator', '', 1, datetime('now')),
			('user-going', 'going@example.com', 'hash', 'Going', 'User', '1991-01-01', 'going', '', 1, datetime('now')),
			('user-not-going', 'notgoing@example.com', 'hash', 'NotGoing', 'User', '1992-01-01', 'notgoing', '', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("seed users: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES ('group-1', 'creator-1', 'Group One', 'Description', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO events (id, group_id, creator_id, title, description, event_date, created_at)
		VALUES ('event-1', 'group-1', 'creator-1', 'Launch', 'Description', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}

	if err := repo.CreateEventRSVP("event-1", "user-going", "going"); err != nil {
		t.Fatalf("create going rsvp: %v", err)
	}
	if err := repo.CreateEventRSVP("event-1", "user-not-going", "not_going"); err != nil {
		t.Fatalf("create not going rsvp: %v", err)
	}

	attendees, err := repo.GetEventAttendees("event-1", "going")
	if err != nil {
		t.Fatalf("get event attendees: %v", err)
	}
	if len(attendees) != 1 {
		t.Fatalf("expected 1 going attendee, got %d", len(attendees))
	}
	if attendees[0].ID != "user-going" {
		t.Fatalf("expected user-going attendee, got %q", attendees[0].ID)
	}
}
