package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"golang.org/x/crypto/bcrypt"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Generate a unique database name per test to ensure isolation.
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate random db name: %v", err)
	}
	dbName := hex.EncodeToString(bytes)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_foreign_keys=on&_journal_mode=WAL", dbName)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// Restrict to 1 connection to prevent concurrency issues and match single-conn behavior.
	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		db.Close()
	})

	if err := runMigrations(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func runMigrations(db *sql.DB) error {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsPath := filepath.Join(basepath, "..", "db", "migrations", "sqlite")

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func TestRegister(t *testing.T) {
	validReq := models.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		DOB:       "1990-01-01",
		Nickname:  "johndoe",
		AboutMe:   "Hello, I am John",
	}

	tests := []struct {
		name         string
		req          models.RegisterRequest
		setupDB      func(t *testing.T, db *sql.DB, users repository.UserRepository)
		assertErr    func(t *testing.T, err error)
		expectResult bool
	}{
		{
			name:         "Success",
			req:          validReq,
			expectResult: true,
		},
		{
			name: "Empty Email",
			req: func() models.RegisterRequest {
				r := validReq
				r.Email = ""
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Invalid Email (no @)",
			req: func() models.RegisterRequest {
				r := validReq
				r.Email = "invalid-email"
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Short Password",
			req: func() models.RegisterRequest {
				r := validReq
				r.Password = "short"
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Empty First Name",
			req: func() models.RegisterRequest {
				r := validReq
				r.FirstName = ""
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Empty Last Name",
			req: func() models.RegisterRequest {
				r := validReq
				r.LastName = ""
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Empty DOB",
			req: func() models.RegisterRequest {
				r := validReq
				r.DOB = ""
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Invalid DOB Format",
			req: func() models.RegisterRequest {
				r := validReq
				r.DOB = "01-01-1990"
				return r
			}(),
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name: "Email Conflict",
			req:  validReq,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				err := users.CreateUser(&models.User{
					ID:        "existing-id",
					Email:     "test@example.com",
					Password:  "hash",
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrConflict) {
					t.Errorf("expected ErrConflict, got %v", err)
				}
			},
		},
		{
			name: "User Repo Create Error",
			req:  validReq,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_, err := db.Exec("DROP TABLE users")
				if err != nil {
					t.Fatalf("failed to drop users table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
		{
			name: "Session Repo Create Error",
			req:  validReq,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_, err := db.Exec("DROP TABLE sessions")
				if err != nil {
					t.Fatalf("failed to drop sessions table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			sessions := repository.NewSessionRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users)
			}

			svc := NewService(users, sessions)
			res, err := svc.Register(tt.req)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil response, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.expectResult {
					if res == nil {
						t.Fatal("expected non-nil response")
					}
					if res.Token == "" {
						t.Error("expected non-empty session token")
					}
					if res.User == nil {
						t.Error("expected non-nil user in response")
					} else {
						if res.User.FirstName != tt.req.FirstName {
							t.Errorf("expected FirstName %q, got %q", tt.req.FirstName, res.User.FirstName)
						}
					}

					// Verify that the user was actually inserted in the DB
					dbUser, err := users.GetUserByEmail(tt.req.Email)
					if err != nil {
						t.Fatalf("expected user to be in DB, got error: %v", err)
					}
					if dbUser.FirstName != tt.req.FirstName {
						t.Errorf("expected DB FirstName %q, got %q", tt.req.FirstName, dbUser.FirstName)
					}
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	const (
		testEmail    = "user@example.com"
		testPassword = "password123"
	)

	// Pre-generate password hash to speed up tests
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcryptCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	tests := []struct {
		name         string
		email        string
		password     string
		setupDB      func(t *testing.T, db *sql.DB, users repository.UserRepository)
		assertErr    func(t *testing.T, err error)
		expectResult bool
	}{
		{
			name:     "Success",
			email:    testEmail,
			password: testPassword,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				err := users.CreateUser(&models.User{
					ID:        "user-123",
					Email:     testEmail,
					Password:  string(hashedPassword),
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
			},
			expectResult: true,
		},
		{
			name:     "Empty Email",
			email:    "",
			password: testPassword,
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "Empty Password",
			email:    testEmail,
			password: "",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrBadInput) {
					t.Errorf("expected ErrBadInput, got %v", err)
				}
			},
		},
		{
			name:     "User Not Found",
			email:    "nonexistent@example.com",
			password: testPassword,
			assertErr: func(t *testing.T, err error) {
				// GetUserByEmail returns ErrUnauthorized ("invalid credentials") when user is not found.
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
			},
		},
		{
			name:     "Invalid Password",
			email:    testEmail,
			password: "wrongpassword",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				err := users.CreateUser(&models.User{
					ID:        "user-123",
					Email:     testEmail,
					Password:  string(hashedPassword),
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
			},
		},
		{
			name:     "User Repo Error",
			email:    testEmail,
			password: testPassword,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				_, err := db.Exec("DROP TABLE users")
				if err != nil {
					t.Fatalf("failed to drop users table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
		{
			name:     "Session Repo Error",
			email:    testEmail,
			password: testPassword,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository) {
				err := users.CreateUser(&models.User{
					ID:        "user-123",
					Email:     testEmail,
					Password:  string(hashedPassword),
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
				_, err = db.Exec("DROP TABLE sessions")
				if err != nil {
					t.Fatalf("failed to drop sessions table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			sessions := repository.NewSessionRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users)
			}

			svc := NewService(users, sessions)
			res, err := svc.Login(tt.email, tt.password)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if res != nil {
					t.Errorf("expected nil response, got %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.expectResult {
					if res == nil {
						t.Fatal("expected non-nil response")
					}
					if res.Token == "" {
						t.Error("expected non-empty session token")
					}
					if res.User == nil {
						t.Error("expected non-nil user in response")
					}
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		setupDB   func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository)
		assertErr func(t *testing.T, err error)
	}{
		{
			name:  "Success",
			token: "token-123",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				err := users.CreateUser(&models.User{
					ID:        "user-123",
					Email:     "user@example.com",
					Password:  "hash",
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
				err = sessions.CreateSession(&models.Session{
					Token:     "token-123",
					UserID:    "user-123",
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
					ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				})
				if err != nil {
					t.Fatalf("failed to seed session: %v", err)
				}
			},
		},
		{
			name:  "Session Repo Error",
			token: "token-123",
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				_, err := db.Exec("DROP TABLE sessions")
				if err != nil {
					t.Fatalf("failed to drop sessions table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			sessions := repository.NewSessionRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, sessions)
			}

			svc := NewService(users, sessions)
			err := svc.Logout(tt.token)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Verify session is deleted
				_, err = sessions.GetSessionByToken(tt.token)
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected session to be deleted (ErrUnauthorized), got %v", err)
				}
			}
		})
	}
}

func TestValidateSession(t *testing.T) {
	const (
		testToken  = "valid-token"
		testUserID = "user-123"
	)

	tests := []struct {
		name          string
		token         string
		setupDB       func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository)
		assertErr     func(t *testing.T, err error)
		expectDeleted bool
	}{
		{
			name:  "Success",
			token: testToken,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				err := users.CreateUser(&models.User{
					ID:        testUserID,
					Email:     "user@example.com",
					Password:  "hash",
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
				err = sessions.CreateSession(&models.Session{
					Token:     testToken,
					UserID:    testUserID,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
					ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				})
				if err != nil {
					t.Fatalf("failed to seed session: %v", err)
				}
			},
		},
		{
			name:  "Empty Token",
			token: "",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
			},
		},
		{
			name:  "Session Not Found",
			token: "nonexistent-token",
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
			},
		},
		{
			name:  "Session Repo Error",
			token: testToken,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				_, err := db.Exec("DROP TABLE sessions")
				if err != nil {
					t.Fatalf("failed to drop sessions table: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
		{
			name:  "Session Expired",
			token: testToken,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				err := users.CreateUser(&models.User{
					ID:        testUserID,
					Email:     "user@example.com",
					Password:  "hash",
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
				err = sessions.CreateSession(&models.Session{
					Token:     testToken,
					UserID:    testUserID,
					CreatedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
					ExpiresAt: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				})
				if err != nil {
					t.Fatalf("failed to seed session: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrUnauthorized) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
			},
			expectDeleted: true,
		},
		{
			name:  "User Not Found",
			token: testToken,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				// Temporarily disable foreign keys to insert a session without a user.
				_, err := db.Exec("PRAGMA foreign_keys = OFF")
				if err != nil {
					t.Fatalf("failed to disable foreign keys: %v", err)
				}
				err = sessions.CreateSession(&models.Session{
					Token:     testToken,
					UserID:    testUserID,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
					ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				})
				if err != nil {
					t.Fatalf("failed to seed session: %v", err)
				}
				_, err = db.Exec("PRAGMA foreign_keys = ON")
				if err != nil {
					t.Fatalf("failed to enable foreign keys: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if !errors.Is(err, apperror.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:  "User Repo Error",
			token: testToken,
			setupDB: func(t *testing.T, db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) {
				err := users.CreateUser(&models.User{
					ID:        testUserID,
					Email:     "user@example.com",
					Password:  "hash",
					FirstName: "Jane",
					LastName:  "Doe",
					DOB:       "1990-01-01",
				})
				if err != nil {
					t.Fatalf("failed to seed user: %v", err)
				}
				err = sessions.CreateSession(&models.Session{
					Token:     testToken,
					UserID:    testUserID,
					CreatedAt: time.Now().UTC().Format(time.RFC3339),
					ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				})
				if err != nil {
					t.Fatalf("failed to seed session: %v", err)
				}
				// Temporarily disable foreign keys, so dropping users does not delete the session via ON DELETE CASCADE
				_, err = db.Exec("PRAGMA foreign_keys = OFF")
				if err != nil {
					t.Fatalf("failed to disable foreign keys: %v", err)
				}
				_, err = db.Exec("DROP TABLE users")
				if err != nil {
					t.Fatalf("failed to drop users table: %v", err)
				}
				_, err = db.Exec("PRAGMA foreign_keys = ON")
				if err != nil {
					t.Fatalf("failed to enable foreign keys: %v", err)
				}
			},
			assertErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected database error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			users := repository.NewUserRepository(db)
			sessions := repository.NewSessionRepository(db)

			if tt.setupDB != nil {
				tt.setupDB(t, db, users, sessions)
			}

			svc := NewService(users, sessions)
			user, err := svc.ValidateSession(tt.token)

			if tt.assertErr != nil {
				tt.assertErr(t, err)
				if user != nil {
					t.Errorf("expected nil user, got %v", user)
				}
				if tt.expectDeleted {
					_, err = sessions.GetSessionByToken(tt.token)
					if !errors.Is(err, apperror.ErrUnauthorized) {
						t.Errorf("expected expired session to be deleted, got error: %v", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if user == nil {
					t.Fatal("expected non-nil user")
				}
				if user.ID != testUserID {
					t.Errorf("expected user ID %q, got %q", testUserID, user.ID)
				}
			}
		})
	}
}
