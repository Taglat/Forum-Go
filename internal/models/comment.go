package models

import "time"

type Comment struct {
	ID      int       // Уникальный идентификатор
	Content string    // Содержимое комментария
	PostID  int       // ID поста к которому привязан комментарий
	UserID  int       // ID автора комментария
	Created time.Time // Дата создания
	Updated time.Time // Дата изменения
	// Данные автора (для JOIN запросов)
	Username string // Имя автора
}
