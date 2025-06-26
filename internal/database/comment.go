package database

import (
	"database/sql"
	"errors"
	"fmt"
	"forum/internal/models"
	"strings"
	"time"
)

var (
	ErrCommentNotFound     = errors.New("комментарий не найден")
	ErrEmptyCommentContent = errors.New("содержимое комментария не может быть пустым")
	ErrLongCommentContent  = errors.New("содержимое комментария не должно превышать 2000 символов")
	ErrCommentCreateFailed = errors.New("ошибка создания комментария")
	ErrCommentUpdateFailed = errors.New("ошибка обновления комментария")
	ErrCommentDeleteFailed = errors.New("ошибка удаления комментария")
	ErrNotCommentAuthor    = errors.New("только автор может изменять комментарий")
)

type CommentService struct {
	db *Database
}

func NewCommentService(db *Database) *CommentService {
	return &CommentService{db: db}
}

// CreateComment создает новый комментарий
func (cs *CommentService) CreateComment(content string, postID, userID int) (*models.Comment, error) {
	if err := cs.validateCommentData(content); err != nil {
		return nil, err
	}

	query := `INSERT INTO comments (content, post_id, user_id, created, updated) 
			  VALUES (?, ?, ?, ?, ?) RETURNING id, created, updated`

	var comment models.Comment
	now := time.Now()

	err := cs.db.DBConn.QueryRow(query, content, postID, userID, now, now).Scan(
		&comment.ID, &comment.Created, &comment.Updated)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCommentCreateFailed, err)
	}

	comment.Content = content
	comment.PostID = postID
	comment.UserID = userID

	return &comment, nil
}

// GetComment получает комментарий по ID с информацией об авторе
func (cs *CommentService) GetComment(id int) (*models.Comment, error) {
	query := `SELECT c.id, c.content, c.post_id, c.user_id, c.created, c.updated, u.username
			  FROM comments c
			  JOIN users u ON c.user_id = u.id
			  WHERE c.id = ?`

	var comment models.Comment
	err := cs.db.DBConn.QueryRow(query, id).Scan(
		&comment.ID, &comment.Content, &comment.PostID, &comment.UserID,
		&comment.Created, &comment.Updated, &comment.Username)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	return &comment, nil
}

// GetPostComments получает все комментарии поста
func (cs *CommentService) GetPostComments(postID int) ([]*models.Comment, error) {
	query := `SELECT c.id, c.content, c.post_id, c.user_id, c.created, c.updated, u.username
			  FROM comments c
			  JOIN users u ON c.user_id = u.id
			  WHERE c.post_id = ?
			  ORDER BY c.created ASC`

	rows, err := cs.db.DBConn.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.PostID, &comment.UserID,
			&comment.Created, &comment.Updated, &comment.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

// GetUserComments получает комментарии конкретного пользователя
func (cs *CommentService) GetUserComments(userID int) ([]*models.Comment, error) {
	query := `SELECT c.id, c.content, c.post_id, c.user_id, c.created, c.updated, u.username
			  FROM comments c
			  JOIN users u ON c.user_id = u.id
			  WHERE c.user_id = ?
			  ORDER BY c.created DESC`

	rows, err := cs.db.DBConn.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var comment models.Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.PostID, &comment.UserID,
			&comment.Created, &comment.Updated, &comment.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

// UpdateComment обновляет комментарий (только автор может изменять)
func (cs *CommentService) UpdateComment(commentID int, content string, userID int) error {
	if err := cs.validateCommentData(content); err != nil {
		return err
	}

	// Проверяем, что пользователь является автором комментария
	if !cs.isCommentAuthor(commentID, userID) {
		return ErrNotCommentAuthor
	}

	query := `UPDATE comments SET content = ?, updated = ? WHERE id = ?`
	result, err := cs.db.DBConn.Exec(query, content, time.Now(), commentID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommentUpdateFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrCommentNotFound
	}

	return nil
}

// DeleteComment удаляет комментарий (только автор может удалять)
func (cs *CommentService) DeleteComment(id int, userID int) error {
	// Проверяем, что пользователь является автором комментария
	if !cs.isCommentAuthor(id, userID) {
		return ErrNotCommentAuthor
	}

	query := `DELETE FROM comments WHERE id = ?`
	result, err := cs.db.DBConn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCommentDeleteFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrCommentNotFound
	}

	return nil
}

// GetCommentsCount получает общее количество комментариев поста
func (cs *CommentService) GetCommentsCount(postID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE post_id = ?`
	err := cs.db.DBConn.QueryRow(query, postID).Scan(&count)
	return count, err
}

// isCommentAuthor проверяет, является ли пользователь автором комментария
func (cs *CommentService) isCommentAuthor(commentID, userID int) bool {
	var authorID int
	query := `SELECT user_id FROM comments WHERE id = ?`
	err := cs.db.DBConn.QueryRow(query, commentID).Scan(&authorID)
	if err != nil {
		return false
	}
	return authorID == userID
}

// validateCommentData валидирует данные комментария
func (cs *CommentService) validateCommentData(content string) error {
	content = strings.TrimSpace(content)

	if len(content) == 0 {
		return ErrEmptyCommentContent
	}
	if len(content) > 2000 {
		return ErrLongCommentContent
	}

	return nil
}
