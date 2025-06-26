package models

import "time"

type Session struct {
	Token   string    // Уникальный токен сессии
	UserID  int       // ID пользователя
	Expires time.Time // Время истечения
	Created time.Time // Время создания
}
