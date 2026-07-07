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
