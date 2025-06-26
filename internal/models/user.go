package models

import "time"

type User struct {
	ID       int       // Уникальный идентификатор
	Username string    // Имя пользователя
	Email    string    // Email (уникален)
	Password []byte    // Хешированный пароль
	Created  time.Time // Дата регистрации
}
