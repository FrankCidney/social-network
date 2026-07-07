package auth

import (
	"fmt"
	"social-network/internal/apperror"
	"social-network/internal/models"
	"social-network/internal/repository"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLen  = 8
	bcryptCost      = 12
	sessionDuration = 7 * 24 * time.Hour
)

type Service interface {
	Register(req models.RegisterRequest) (*models.AuthResponse, error)
	Login(email, password string) (*models.AuthResponse, error) // login creates a session
	Logout(sessionToken string) error
	ValidateSession(token string) (*models.User, error) // called by middleware on every request
}

type service struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
}

func NewService(users repository.UserRepository, sessions repository.SessionRepository) Service {
	return &service{users: users, sessions: sessions}
}

func (s *service) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		ID:        uuid.Must(uuid.NewV4()).String(),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Password:  string(hash),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		DOB:       req.DOB,
		Nickname:  strings.TrimSpace(req.Nickname),
		AboutMe:   strings.TrimSpace(req.AboutMe),
		IsPublic:  true, // chose to set new accounts to public by default; editable after logging in
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.users.CreateUser(user); err != nil {
		return nil, err
	}

	session, err := s.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		User:  user.ToPublic(),
		Token: session.Token,
	}, nil
}

func (s *service) Login(email, password string) (*models.AuthResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil, apperror.BadInput("email and password are required")
	}

	user, err := s.users.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	session, err := s.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		User:  user.ToPublic(),
		Token: session.Token,
	}, nil
}

func (s *service) Logout(sessionToken string) error {
	return s.sessions.DeleteSession(sessionToken)
}

func (s *service) ValidateSession(token string) (*models.User, error) {
	if token == "" {
		return nil, apperror.Unauthorized("missing session token")
	}

	session, err := s.sessions.GetSessionByToken(token)
	if err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil {
		return nil, apperror.Internal("malformed session expiry")
	}

	if time.Now().UTC().After(expiresAt) {
		_ = s.sessions.DeleteSession(token)
		return nil, apperror.Unauthorized("session expired")
	}

	user, err := s.users.GetUserByID(session.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func validateRegisterRequest(req models.RegisterRequest) error {
	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	switch {
	case req.Email == "":
		return apperror.BadInput("email is required")
	case !strings.Contains(req.Email, "@"):
		return apperror.BadInput("invalid email address")
	case len(req.Password) < minPasswordLen:
		return apperror.BadInput(fmt.Sprintf("passwordmust be at least %d characters", minPasswordLen))
	case req.FirstName == "":
		return apperror.BadInput("first name is required")
	case req.LastName == "":
		return apperror.BadInput("last name is required")
	case req.DOB == "":
		return apperror.BadInput("date of birth is required")
	}

	if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
		return apperror.BadInput("date of birth must be in YYYY-MM-DD format")
	}

	return nil
}

func (s *service) createSession(userID string) (*models.Session, error) {
	token := uuid.Must(uuid.NewV4()).String()

	now := time.Now().UTC()
	session := &models.Session{
		Token:     token,
		UserID:    userID,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(sessionDuration).Format(time.RFC3339),
	}

	if err := s.sessions.CreateSession(session); err != nil {
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return session, nil
}
