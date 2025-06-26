package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"forum/internal/models"
	"time"
)

var (
	ErrSessionNotFound = errors.New("сессия не найдена")
	ErrSessionExpired  = errors.New("сессия истекла")
	ErrTokenGeneration = errors.New("ошибка генерации токена")
	ErrSessionCreation = errors.New("ошибка создания сессии")
	ErrSessionDeletion = errors.New("ошибка удаления сессии")
)

const (
	// Время жизни сессии - 24 часа
	SessionDuration = 24 * time.Hour
	// Длина токена в байтах (32 байта = 64 символа в hex)
	TokenLength = 32
)

type SessionService struct {
	db *Database
}

func NewSessionService(db *Database) *SessionService {
	return &SessionService{db: db}
}

func (ss *SessionService) CreateSession(userID int) (*models.Session, error) {
	// Удаляем все существующие сессии пользователя
	if err := ss.DeleteUserSessions(userID); err != nil {
		return nil, fmt.Errorf("ошибка удаления старых сессий: %v", err)
	}

	// Генерируем уникальный токен
	token, err := ss.generateToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenGeneration, err)
	}

	now := time.Now()
	expires := now.Add(SessionDuration)

	query := `INSERT INTO sessions (token, user_id, expires, created) VALUES (?, ?, ?, ?)`
	_, err = ss.db.DBConn.Exec(query, token, userID, expires, now)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionCreation, err)
	}

	return &models.Session{
		Token:   token,
		UserID:  userID,
		Expires: expires,
		Created: now,
	}, nil
}

// GetSession получает сессию по токену и проверяет срок действия
func (ss *SessionService) GetSession(token string) (*models.Session, error) {
	var session models.Session

	query := `SELECT token, user_id, expires, created FROM sessions WHERE token = ?`
	err := ss.db.DBConn.QueryRow(query, token).Scan(
		&session.Token,
		&session.UserID,
		&session.Expires,
		&session.Created,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// Проверяем, не истекла ли сессия
	if time.Now().After(session.Expires) {
		// Удаляем истекшую сессию
		ss.DeleteSession(token)
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// GetUserBySession получает пользователя по токену сессии
func (ss *SessionService) GetUserBySession(token string) (*models.User, error) {
	session, err := ss.GetSession(token)
	if err != nil {
		return nil, err
	}

	var user models.User
	query := `SELECT id, username, email, password, created FROM users WHERE id = ?`
	err = ss.db.DBConn.QueryRow(query, session.UserID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Created,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Удаляем сессию, если пользователь не найден
			ss.DeleteSession(token)
			return nil, errors.New("пользователь не найден")
		}
		return nil, err
	}

	return &user, nil
}

// DeleteSession удаляет сессию по токену
func (ss *SessionService) DeleteSession(token string) error {
	query := `DELETE FROM sessions WHERE token = ?`
	result, err := ss.db.DBConn.Exec(query, token)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionDeletion, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteUserSessions удаляет все сессии пользователя
func (ss *SessionService) DeleteUserSessions(userID int) error {
	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := ss.db.DBConn.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionDeletion, err)
	}
	return nil
}

// CleanupExpiredSessions удаляет истекшие сессии
func (ss *SessionService) CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires < ?`
	_, err := ss.db.DBConn.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("ошибка очистки истекших сессий: %v", err)
	}
	return nil
}

// generateToken генерирует криптографически стойкий токен
func (ss *SessionService) generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
