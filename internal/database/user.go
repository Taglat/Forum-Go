package database

import (
	"database/sql"
	"errors"
	"fmt"
	"forum/internal/models"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameExists     = errors.New("пользователь с таким именем уже существует")
	ErrEmailExists        = errors.New("пользователь с таким email уже существует")
	ErrEmptyEmail         = errors.New("email не может быть пустым")
	ErrLongEmail          = errors.New("email не должен превышать 255 символов")
	ErrInvalidUsername    = errors.New("имя пользователя может содержать только буквы, цифры, подчеркивание и дефис")
	ErrShortUsername      = errors.New("имя пользователя должно содержать минимум 3 символа")
	ErrLongUsername       = errors.New("имя пользователя не должно превышать 50 символов")
	ErrShortPassword      = errors.New("пароль должен содержать минимум 6 символов")
	ErrLongPassword       = errors.New("пароль не должен превышать 128 символов")
	ErrPasswordHashFailed = errors.New("ошибка хеширования пароля")
	ErrUserCreateFailed   = errors.New("ошибка создания пользователя")
	ErrEmailNotFound      = errors.New("пользователь с таким email не найден")
	ErrIncorrectPassword  = errors.New("неверный пароль")
)

type UserService struct {
	db *Database
}

func NewUserService(db *Database) *UserService {
	return &UserService{db: db}
}

func (us *UserService) CreateUser(username, email, password string) (*models.User, error) {
	// Валидация входных данных
	if err := us.validateUserData(username, email, password); err != nil {
		return nil, err
	}

	// Проверяем уникальность username и email
	if err := us.checkUserUniqueness(username, email); err != nil {
		return nil, err
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPasswordHashFailed, err)
	}

	// SQL запрос для вставки пользователя
	query := `INSERT INTO users (username, email, password, created) 
   		  VALUES (?, ?, ?, ?) RETURNING id, created`

	var user models.User
	now := time.Now()

	// Выполняем запрос и получаем ID созданного пользователя
	err = us.db.DBConn.QueryRow(query, username, email, hashedPassword, now).Scan(&user.ID, &user.Created)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserCreateFailed, err)
	}

	// Заполняем остальные поля структуры
	user.Username = username
	user.Email = email
	user.Password = hashedPassword

	return &user, nil
}

func (us *UserService) VerifyUser(email, password string) (int, string, error) {
	var id int
	var username string
	var hashedPassword []byte

	query := `SELECT id, username, password FROM users WHERE email = ?`
	err := us.db.DBConn.QueryRow(query, email).Scan(&id, &username, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", ErrEmailNotFound
		}
		return 0, "", err
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return 0, "", ErrIncorrectPassword
	}

	return id, username, nil
}

// checkUserUniqueness проверяет уникальность username и email
func (us *UserService) checkUserUniqueness(username, email string) error {
	// Проверяем username
	query := `SELECT 1 FROM users WHERE username = ?`
	var exists int
	err := us.db.DBConn.QueryRow(query, username).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrUsernameExists
		}
		return fmt.Errorf("ошибка проверки уникальности username: %v", err)
	}

	// Проверяем email
	query = `SELECT 1 FROM users WHERE email = ?`
	err = us.db.DBConn.QueryRow(query, email).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrEmailExists
		}
		return fmt.Errorf("ошибка проверки уникальности email: %v", err)
	}

	return nil
}

// validateUserData валидирует все данные пользователя
func (us *UserService) validateUserData(username, email, password string) error {
	if err := us.validateUsername(username); err != nil {
		return err
	}
	if err := us.validateEmail(email); err != nil {
		return err
	}
	if err := us.validatePassword(password); err != nil {
		return err
	}
	return nil
}

// validateEmail валидирует email адрес
func (us *UserService) validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return ErrEmptyEmail
	}
	if len(email) > 255 {
		return ErrLongEmail
	}

	return nil
}

// validateUsername валидирует имя пользователя
func (us *UserService) validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return ErrShortUsername
	}
	if len(username) > 50 {
		return ErrLongUsername
	}

	// Проверяем на допустимые символы (буквы, цифры, подчеркивание, дефис)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username)
	if !matched {
		return ErrInvalidUsername
	}

	return nil
}

func (us *UserService) validatePassword(password string) error {
	if len(password) < 6 {
		return ErrShortPassword
	}
	if len(password) > 128 {
		return ErrLongPassword
	}
	return nil
}
