package models

import "time"

type Like struct {
	ID        int       // Уникальный идентификатор
	UserID    int       // ID пользователя
	PostID    *int      // ID поста (если лайк на пост)
	CommentID *int      // ID комментария (если лайк на комментарий)
	IsDislike bool      // true для дизлайка, false для лайка
	Created   time.Time // Дата создания
}

type LikeStats struct {
	Likes    int // Количество лайков
	Dislikes int // Количество дизлайков
}
