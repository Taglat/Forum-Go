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
	ErrPostNotFound     = errors.New("пост не найден")
	ErrEmptyTitle       = errors.New("заголовок не может быть пустым")
	ErrLongTitle        = errors.New("заголовок не должен превышать 255 символов")
	ErrEmptyContent     = errors.New("содержимое поста не может быть пустым")
	ErrLongContent      = errors.New("содержимое поста не должно превышать 10000 символов")
	ErrPostCreateFailed = errors.New("ошибка создания поста")
	ErrPostUpdateFailed = errors.New("ошибка обновления поста")
	ErrPostDeleteFailed = errors.New("ошибка удаления поста")
	ErrNotPostAuthor    = errors.New("только автор может изменять пост")
)

type PostService struct {
	db *Database
}

func NewPostService(db *Database) *PostService {
	return &PostService{db: db}
}

// CreatePost создает новый пост
func (ps *PostService) CreatePost(title, content string, userID int) (*models.Post, error) {
	if err := ps.validatePostData(title, content); err != nil {
		return nil, err
	}

	query := `INSERT INTO posts (title, content, user_id, created, updated) 
			  VALUES (?, ?, ?, ?, ?) RETURNING id, created, updated`

	var post models.Post
	now := time.Now()

	err := ps.db.DBConn.QueryRow(query, title, content, userID, now, now).Scan(
		&post.ID, &post.Created, &post.Updated)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPostCreateFailed, err)
	}

	post.Title = title
	post.Content = content
	post.UserID = userID

	return &post, nil
}

// GetPost получает пост по ID с информацией об авторе
func (ps *PostService) GetPost(id int) (*models.Post, error) {
	query := `SELECT p.id, p.title, p.content, p.user_id, p.created, p.updated, u.username
			  FROM posts p
			  JOIN users u ON p.user_id = u.id
			  WHERE p.id = ?`

	var post models.Post
	err := ps.db.DBConn.QueryRow(query, id).Scan(
		&post.ID, &post.Title, &post.Content, &post.UserID,
		&post.Created, &post.Updated, &post.Username)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	return &post, nil
}

// GetAllPosts получает все посты с пагинацией
func (ps *PostService) GetAllPosts(limit, offset int) ([]*models.Post, error) {
	query := `SELECT p.id, p.title, p.content, p.user_id, p.created, p.updated, u.username
			  FROM posts p
			  JOIN users u ON p.user_id = u.id
			  ORDER BY p.created DESC
			  LIMIT ? OFFSET ?`

	rows, err := ps.db.DBConn.Query(query, limit, offset)
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

// GetUserPosts получает посты конкретного пользователя
func (ps *PostService) GetUserPosts(userID int) ([]*models.Post, error) {
	query := `SELECT p.id, p.title, p.content, p.user_id, p.created, p.updated, u.username
			  FROM posts p
			  JOIN users u ON p.user_id = u.id
			  WHERE p.user_id = ?
			  ORDER BY p.created DESC`

	rows, err := ps.db.DBConn.Query(query, userID)
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

// UpdatePost обновляет пост (только автор может изменять)
func (ps *PostService) UpdatePost(id int, title, content string, userID int) error {
	if err := ps.validatePostData(title, content); err != nil {
		return err
	}

	// Проверяем, что пользователь является автором поста
	if !ps.isPostAuthor(id, userID) {
		return ErrNotPostAuthor
	}

	query := `UPDATE posts SET title = ?, content = ?, updated = ? WHERE id = ?`
	_, err := ps.db.DBConn.Exec(query, title, content, time.Now(), id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostUpdateFailed, err)
	}

	return nil
}

// DeletePost удаляет пост (только автор может удалять)
func (ps *PostService) DeletePost(id int, userID int) error {
	// Проверяем, что пользователь является автором поста
	if !ps.isPostAuthor(id, userID) {
		return ErrNotPostAuthor
	}

	query := `DELETE FROM posts WHERE id = ?`
	result, err := ps.db.DBConn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPostDeleteFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrPostNotFound
	}

	return nil
}

// GetPostsCount получает общее количество постов
func (ps *PostService) GetPostsCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM posts`
	err := ps.db.DBConn.QueryRow(query).Scan(&count)
	return count, err
}

// isPostAuthor проверяет, является ли пользователь автором поста
func (ps *PostService) isPostAuthor(postID, userID int) bool {
	var authorID int
	query := `SELECT user_id FROM posts WHERE id = ?`
	err := ps.db.DBConn.QueryRow(query, postID).Scan(&authorID)
	if err != nil {
		return false
	}
	return authorID == userID
}

// validatePostData валидирует данные поста
func (ps *PostService) validatePostData(title, content string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if len(title) == 0 {
		return ErrEmptyTitle
	}
	if len(title) > 255 {
		return ErrLongTitle
	}
	if len(content) == 0 {
		return ErrEmptyContent
	}
	if len(content) > 10000 {
		return ErrLongContent
	}

	return nil
}
