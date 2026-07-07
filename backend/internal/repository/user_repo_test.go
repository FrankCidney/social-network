package repository

import (
	"errors"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"testing"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo UserRepository)
		user    *models.User
		wantErr error
	}{
		{
			name:  "success_new_user",
			setup: func(t *testing.T, repo UserRepository) {},
			user: &models.User{
				ID:         "user-1",
				Email:      "user1@example.com",
				Password:   "hashpassword",
				FirstName:  "John",
				LastName:   "Doe",
				DOB:        "1990-01-01T00:00:00Z",
				Nickname:   "johndoe",
				AboutMe:    "Hello, I am John",
				AvatarPath: "/path/to/avatar.png",
				IsPublic:   true,
				CreatedAt:  "2026-06-30T12:00:00Z",
			},
			wantErr: nil,
		},
		{
			name: "conflict_duplicate_email",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:        "user-1",
					Email:     "user1@example.com",
					FirstName: "John",
					LastName:  "Doe",
					DOB:       "1990-01-01T00:00:00Z",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			user: &models.User{
				ID:        "user-2",
				Email:     "user1@example.com", // duplicate email
				FirstName: "Jane",
				LastName:  "Doe",
				DOB:       "1992-02-02T00:00:00Z",
			},
			wantErr: apperror.ErrConflict,
		},
		{
			name: "conflict_duplicate_id",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:        "user-1",
					Email:     "user1@example.com",
					FirstName: "John",
					LastName:  "Doe",
					DOB:       "1990-01-01T00:00:00Z",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			user: &models.User{
				ID:        "user-1", // duplicate ID
				Email:     "user2@example.com",
				FirstName: "Jane",
				LastName:  "Doe",
				DOB:       "1992-02-02T00:00:00Z",
			},
			wantErr: apperror.ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			err := repo.CreateUser(tt.user)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Verify fields
				got, err := repo.GetUserByID(tt.user.ID)
				if err != nil {
					t.Errorf("failed to retrieve created user: %v", err)
				}
				if got.ID != tt.user.ID ||
					got.Email != tt.user.Email ||
					got.Password != tt.user.Password ||
					got.FirstName != tt.user.FirstName ||
					got.LastName != tt.user.LastName ||
					got.DOB != tt.user.DOB ||
					got.Nickname != tt.user.Nickname ||
					got.AboutMe != tt.user.AboutMe ||
					got.AvatarPath != tt.user.AvatarPath ||
					got.IsPublic != tt.user.IsPublic ||
					got.CreatedAt != tt.user.CreatedAt {
					t.Errorf("retrieved user %+v does not match created user %+v", got, tt.user)
				}
			}
		})
	}
}

func TestGetUserByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo UserRepository)
		id      string
		want    *models.User
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:         "user-1",
					Email:      "user1@example.com",
					Password:   "hash",
					FirstName:  "John",
					LastName:   "Doe",
					DOB:        "1990-01-01T00:00:00Z",
					Nickname:   "johndoe",
					AboutMe:    "About me",
					AvatarPath: "/avatar.png",
					IsPublic:   true,
					CreatedAt:  "2026-06-30T12:00:00Z",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			id: "user-1",
			want: &models.User{
				ID:         "user-1",
				Email:      "user1@example.com",
				Password:   "hash",
				FirstName:  "John",
				LastName:   "Doe",
				DOB:        "1990-01-01T00:00:00Z",
				Nickname:   "johndoe",
				AboutMe:    "About me",
				AvatarPath: "/avatar.png",
				IsPublic:   true,
				CreatedAt:  "2026-06-30T12:00:00Z",
			},
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, repo UserRepository) {},
			id:      "non-existent",
			want:    nil,
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			got, err := repo.GetUserByID(tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if got.ID != tt.want.ID ||
					got.Email != tt.want.Email ||
					got.Password != tt.want.Password ||
					got.FirstName != tt.want.FirstName ||
					got.LastName != tt.want.LastName ||
					got.DOB != tt.want.DOB ||
					got.Nickname != tt.want.Nickname ||
					got.AboutMe != tt.want.AboutMe ||
					got.AvatarPath != tt.want.AvatarPath ||
					got.IsPublic != tt.want.IsPublic ||
					got.CreatedAt != tt.want.CreatedAt {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo UserRepository)
		email   string
		want    *models.User
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:         "user-1",
					Email:      "user1@example.com",
					Password:   "hash",
					FirstName:  "John",
					LastName:   "Doe",
					DOB:        "1990-01-01T00:00:00Z",
					Nickname:   "johndoe",
					AboutMe:    "About me",
					AvatarPath: "/avatar.png",
					IsPublic:   true,
					CreatedAt:  "2026-06-30T12:00:00Z",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			email: "user1@example.com",
			want: &models.User{
				ID:         "user-1",
				Email:      "user1@example.com",
				Password:   "hash",
				FirstName:  "John",
				LastName:   "Doe",
				DOB:        "1990-01-01T00:00:00Z",
				Nickname:   "johndoe",
				AboutMe:    "About me",
				AvatarPath: "/avatar.png",
				IsPublic:   true,
				CreatedAt:  "2026-06-30T12:00:00Z",
			},
			wantErr: nil,
		},
		{
			name:    "not_found_returns_unauthorized",
			setup:   func(t *testing.T, repo UserRepository) {},
			email:   "nonexistent@example.com",
			want:    nil,
			wantErr: apperror.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			got, err := repo.GetUserByEmail(tt.email)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if got.ID != tt.want.ID ||
					got.Email != tt.want.Email ||
					got.Password != tt.want.Password ||
					got.FirstName != tt.want.FirstName ||
					got.LastName != tt.want.LastName ||
					got.DOB != tt.want.DOB ||
					got.Nickname != tt.want.Nickname ||
					got.AboutMe != tt.want.AboutMe ||
					got.AvatarPath != tt.want.AvatarPath ||
					got.IsPublic != tt.want.IsPublic ||
					got.CreatedAt != tt.want.CreatedAt {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo UserRepository)
		user    *models.User
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:        "user-1",
					Email:     "user1@example.com",
					FirstName: "John",
					LastName:  "Doe",
					DOB:       "1990-01-01T00:00:00Z",
					Nickname:  "johndoe",
					AboutMe:   "original about me",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			user: &models.User{
				ID:        "user-1",
				FirstName: "Johnny",
				LastName:  "Smith",
				DOB:       "1991-11-11T00:00:00Z",
				Nickname:  "jsmith",
				AboutMe:   "updated about me",
			},
			wantErr: nil,
		},
		{
			name:  "not_found",
			setup: func(t *testing.T, repo UserRepository) {},
			user: &models.User{
				ID:        "non-existent",
				FirstName: "Johnny",
				LastName:  "Smith",
				DOB:       "1991-11-11T00:00:00Z",
				Nickname:  "jsmith",
				AboutMe:   "updated about me",
			},
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			err := repo.UpdateUser(tt.user)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Verify the user was updated
				got, err := repo.GetUserByID(tt.user.ID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if got.FirstName != tt.user.FirstName ||
					got.LastName != tt.user.LastName ||
					got.DOB != tt.user.DOB ||
					got.Nickname != tt.user.Nickname ||
					got.AboutMe != tt.user.AboutMe {
					t.Errorf("user not updated correctly: got %+v", got)
				}
			}
		})
	}
}

func TestSetProfileVisibility(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, repo UserRepository)
		userID   string
		isPublic bool
		wantErr  error
	}{
		{
			name: "success_set_false_to_true",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:       "user-1",
					Email:    "user1@example.com",
					IsPublic: false,
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			userID:   "user-1",
			isPublic: true,
			wantErr:  nil,
		},
		{
			name: "success_set_true_to_false",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:       "user-1",
					Email:    "user1@example.com",
					IsPublic: true,
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			userID:   "user-1",
			isPublic: false,
			wantErr:  nil,
		},
		{
			name:     "not_found",
			setup:    func(t *testing.T, repo UserRepository) {},
			userID:   "non-existent",
			isPublic: true,
			wantErr:  apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			err := repo.SetProfileVisibility(tt.userID, tt.isPublic)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				got, err := repo.GetUserByID(tt.userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if got.IsPublic != tt.isPublic {
					t.Errorf("expected isPublic to be %v, got %v", tt.isPublic, got.IsPublic)
				}
			}
		})
	}
}

func TestUpdateAvatarPath(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, repo UserRepository)
		userID  string
		path    string
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, repo UserRepository) {
				u := &models.User{
					ID:         "user-1",
					Email:      "user1@example.com",
					AvatarPath: "/old.png",
				}
				if err := repo.CreateUser(u); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			userID:  "user-1",
			path:    "/new.png",
			wantErr: nil,
		},
		{
			name:    "not_found",
			setup:   func(t *testing.T, repo UserRepository) {},
			userID:  "non-existent",
			path:    "/new.png",
			wantErr: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			repo := NewUserRepository(db)
			tt.setup(t, repo)

			err := repo.UpdateAvatarPath(tt.userID, tt.path)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error kind %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				got, err := repo.GetUserByID(tt.userID)
				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}
				if got.AvatarPath != tt.path {
					t.Errorf("expected avatar path to be %q, got %q", tt.path, got.AvatarPath)
				}
			}
		})
	}
}
