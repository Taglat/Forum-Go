package models

import "time"

type Post struct {
	ID      int       // Уникальный идентификатор
	Title   string    // Заголовок поста
	Content string    // Содержимое поста
	UserID  int       // ID автора
	Created time.Time // Дата создания
	Updated time.Time // Дата изменения
	// Данные автора (для JOIN запросов)
	Username string // Имя автора
}
