package user

import (
	"testing"

	"social-network/internal/models"
	"social-network/internal/repository"
)

func TestSearchUsers(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	followRepo := repository.NewFollowRepository(db)
	service := NewService(userRepo, followRepo)

	users := []*models.User{
		{ID: "viewer-1", Email: "viewer@example.com", Password: "hash", FirstName: "Viewer", LastName: "User", DOB: "1990-01-01", IsPublic: true},
		{ID: "alice-1", Email: "alice@example.com", Password: "hash", FirstName: "Alice", LastName: "Jones", Nickname: "ally", DOB: "1991-01-01", IsPublic: true},
		{ID: "alice-2", Email: "alice2@example.com", Password: "hash", FirstName: "Alicia", LastName: "Stone", Nickname: "ali", DOB: "1992-01-01", IsPublic: false},
		{ID: "bob-1", Email: "bob@example.com", Password: "hash", FirstName: "Bob", LastName: "Taylor", Nickname: "bobby", DOB: "1993-01-01", IsPublic: true},
	}

	for _, user := range users {
		if err := userRepo.CreateUser(user); err != nil {
			t.Fatalf("create user %s: %v", user.ID, err)
		}
	}

	_, err := db.Exec(`
		INSERT INTO groups (id, creator_id, title, description, created_at)
		VALUES ('group-1', 'viewer-1', 'Group One', 'Description', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO group_members (group_id, user_id, status, created_at)
		VALUES
			('group-1', 'viewer-1', 'accepted', datetime('now')),
			('group-1', 'alice-1', 'accepted', datetime('now')),
			('group-1', 'alice-2', 'invited', datetime('now'))`)
	if err != nil {
		t.Fatalf("seed group members: %v", err)
	}

	results, err := service.SearchUsers("viewer-1", "ali", 10, "group-1")
	if err != nil {
		t.Fatalf("search users: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected excluded group users to be filtered out, got %d result(s)", len(results))
	}

	results, err = service.SearchUsers("viewer-1", "bo", 10, "group-1")
	if err != nil {
		t.Fatalf("search users: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].ID != "bob-1" {
		t.Fatalf("expected bob-1, got %s", results[0].ID)
	}

	results, err = service.SearchUsers("viewer-1", "   ", 10, "group-1")
	if err != nil {
		t.Fatalf("expected blank query to return discoverable users without error: %v", err)
	}

	if len(results) != 1 || results[0].ID != "bob-1" {
		t.Fatalf("expected blank query to return available users, got %+v", results)
	}
}
