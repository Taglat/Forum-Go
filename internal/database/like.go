package database

import (
	"database/sql"
	"errors"
	"fmt"
	"forum/internal/models"
	"time"
)

var (
	ErrLikeNotFound      = errors.New("лайк не найден")
	ErrLikeCreateFailed  = errors.New("ошибка создания лайка")
	ErrLikeDeleteFailed  = errors.New("ошибка удаления лайка")
	ErrInvalidLikeTarget = errors.New("необходимо указать либо post_id, либо comment_id")
	ErrLikeAlreadyExists = errors.New("пользователь уже поставил лайк/дизлайк")
)

type LikeService struct {
	db *Database
}

func NewLikeService(db *Database) *LikeService {
	return &LikeService{db: db}
}

// LikePost ставит лайк посту
func (ls *LikeService) LikePost(postID, userID int) error {
	return ls.createLike(userID, &postID, nil, false)
}

// DislikePost ставит дизлайк посту
func (ls *LikeService) DislikePost(postID, userID int) error {
	return ls.createLike(userID, &postID, nil, true)
}

// LikeComment ставит лайк комментарию
func (ls *LikeService) LikeComment(commentID, userID int) error {
	return ls.createLike(userID, nil, &commentID, false)
}

// DislikeComment ставит дизлайк комментарию
func (ls *LikeService) DislikeComment(commentID, userID int) error {
	return ls.createLike(userID, nil, &commentID, true)
}

// RemovePostLike удаляет лайк/дизлайк с поста
func (ls *LikeService) RemovePostLike(postID, userID int) error {
	return ls.removeLike(userID, &postID, nil)
}

// RemoveCommentLike удаляет лайк/дизлайк с комментария
func (ls *LikeService) RemoveCommentLike(commentID, userID int) error {
	return ls.removeLike(userID, nil, &commentID)
}

// GetPostLikeStats получает статистику лайков поста
func (ls *LikeService) GetPostLikeStats(postID int) (*models.LikeStats, error) {
	return ls.getLikeStats(&postID, nil)
}

// GetCommentLikeStats получает статистику лайков комментария
func (ls *LikeService) GetCommentLikeStats(commentID int) (*models.LikeStats, error) {
	return ls.getLikeStats(nil, &commentID)
}

// GetUserPostLike получает лайк пользователя на пост (если есть)
func (ls *LikeService) GetUserPostLike(postID, userID int) (*models.Like, error) {
	return ls.getUserLike(userID, &postID, nil)
}

// GetUserCommentLike получает лайк пользователя на комментарий (если есть)
func (ls *LikeService) GetUserCommentLike(commentID, userID int) (*models.Like, error) {
	return ls.getUserLike(userID, nil, &commentID)
}

// GetUserLikedPosts получает посты, которые лайкнул пользователь
func (ls *LikeService) GetUserLikedPosts(userID int) ([]*models.Post, error) {
	query := `SELECT p.id, p.title, p.content, p.user_id, p.created, p.updated, u.username
			  FROM posts p
			  JOIN users u ON p.user_id = u.id
			  JOIN likes l ON p.id = l.post_id
			  WHERE l.user_id = ? AND l.is_dislike = false
			  ORDER BY l.created DESC`

	rows, err := ls.db.DBConn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID,
			&post.Created, &post.Updated, &post.Username)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// createLike создает лайк/дизлайк
func (ls *LikeService) createLike(userID int, postID, commentID *int, isDislike bool) error {
	if postID == nil && commentID == nil {
		return ErrInvalidLikeTarget
	}

	// Проверяем, есть ли уже лайк/дизлайк от этого пользователя
	existingLike, err := ls.getUserLike(userID, postID, commentID)
	if err != nil && err != ErrLikeNotFound {
		return err
	}

	if existingLike != nil {
		// Если уже есть такой же лайк/дизлайк, ничего не делаем
		if existingLike.IsDislike == isDislike {
			return ErrLikeAlreadyExists
		}
		// Если есть противоположный лайк/дизлайк, обновляем его
		return ls.updateLike(existingLike.ID, isDislike)
	}

	// Создаем новый лайк/дизлайк
	query := `INSERT INTO likes (user_id, post_id, comment_id, is_dislike, created) 
			  VALUES (?, ?, ?, ?, ?)`

	_, err = ls.db.DBConn.Exec(query, userID, postID, commentID, isDislike, time.Now())
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLikeCreateFailed, err)
	}

	return nil
}

// removeLike удаляет лайк/дизлайк
func (ls *LikeService) removeLike(userID int, postID, commentID *int) error {
	if postID == nil && commentID == nil {
		return ErrInvalidLikeTarget
	}

	var query string
	var args []interface{}

	if postID != nil {
		query = `DELETE FROM likes WHERE user_id = ? AND post_id = ?`
		args = []interface{}{userID, *postID}
	} else {
		query = `DELETE FROM likes WHERE user_id = ? AND comment_id = ?`
		args = []interface{}{userID, *commentID}
	}

	result, err := ls.db.DBConn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLikeDeleteFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrLikeNotFound
	}

	return nil
}

// updateLike обновляет существующий лайк
func (ls *LikeService) updateLike(likeID int, isDislike bool) error {
	query := `UPDATE likes SET is_dislike = ? WHERE id = ?`
	_, err := ls.db.DBConn.Exec(query, isDislike, likeID)
	if err != nil {
		return fmt.Errorf("ошибка обновления лайка: %v", err)
	}
	return nil
}

// getLikeStats получает статистику лайков/дизлайков
func (ls *LikeService) getLikeStats(postID, commentID *int) (*models.LikeStats, error) {
	if postID == nil && commentID == nil {
		return nil, ErrInvalidLikeTarget
	}

	var query string
	var args []interface{}

	if postID != nil {
		query = `SELECT 
				   COUNT(CASE WHEN is_dislike = false THEN 1 END) as likes,
				   COUNT(CASE WHEN is_dislike = true THEN 1 END) as dislikes
				 FROM likes WHERE post_id = ?`
		args = []interface{}{*postID}
	} else {
		query = `SELECT 
				   COUNT(CASE WHEN is_dislike = false THEN 1 END) as likes,
				   COUNT(CASE WHEN is_dislike = true THEN 1 END) as dislikes
				 FROM likes WHERE comment_id = ?`
		args = []interface{}{*commentID}
	}

	var stats models.LikeStats
	err := ls.db.DBConn.QueryRow(query, args...).Scan(&stats.Likes, &stats.Dislikes)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// getUserLike получает лайк пользователя
func (ls *LikeService) getUserLike(userID int, postID, commentID *int) (*models.Like, error) {
	if postID == nil && commentID == nil {
		return nil, ErrInvalidLikeTarget
	}

	var query string
	var args []interface{}

	if postID != nil {
		query = `SELECT id, user_id, post_id, comment_id, is_dislike, created 
				 FROM likes WHERE user_id = ? AND post_id = ?`
		args = []interface{}{userID, *postID}
	} else {
		query = `SELECT id, user_id, post_id, comment_id, is_dislike, created 
				 FROM likes WHERE user_id = ? AND comment_id = ?`
		args = []interface{}{userID, *commentID}
	}

	var like models.Like
	var nullablePostID, nullableCommentID sql.NullInt64

	err := ls.db.DBConn.QueryRow(query, args...).Scan(
		&like.ID, &like.UserID, &nullablePostID, &nullableCommentID,
		&like.IsDislike, &like.Created)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLikeNotFound
		}
		return nil, err
	}

	if nullablePostID.Valid {
		postIDValue := int(nullablePostID.Int64)
		like.PostID = &postIDValue
	}
	if nullableCommentID.Valid {
		commentIDValue := int(nullableCommentID.Int64)
		like.CommentID = &commentIDValue
	}

	return &like, nil
}
