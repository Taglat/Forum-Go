package database

import (
	"database/sql"
	"errors"
	"fmt"
	"forum/internal/models"
	"regexp"
	"strings"
	"time"
)

var (
	ErrCategoryNotFound     = errors.New("категория не найдена")
	ErrCategoryExists       = errors.New("категория с таким именем уже существует")
	ErrSlugExists           = errors.New("категория с таким slug уже существует")
	ErrEmptyCategoryName    = errors.New("название категории не может быть пустым")
	ErrLongCategoryName     = errors.New("название категории не должно превышать 100 символов")
	ErrEmptySlug            = errors.New("slug не может быть пустым")
	ErrLongSlug             = errors.New("slug не должен превышать 100 символов")
	ErrInvalidSlug          = errors.New("slug может содержать только строчные буквы, цифры и дефисы")
	ErrLongDescription      = errors.New("описание не должно превышать 500 символов")
	ErrCategoryCreateFailed = errors.New("ошибка создания категории")
	ErrCategoryUpdateFailed = errors.New("ошибка обновления категории")
	ErrCategoryDeleteFailed = errors.New("ошибка удаления категории")
)

type CategoryService struct {
	db *Database
}

func NewCategoryService(db *Database) *CategoryService {
	return &CategoryService{db: db}
}

// CreateCategory создает новую категорию
func (cs *CategoryService) CreateCategory(name, slug, description string) (*models.Category, error) {
	if err := cs.validateCategoryData(name, slug, description); err != nil {
		return nil, err
	}

	if err := cs.checkCategoryUniqueness(name, slug); err != nil {
		return nil, err
	}

	query := `INSERT INTO categories (name, slug, description, created) 
			  VALUES (?, ?, ?, ?) RETURNING id, created`

	var category models.Category
	now := time.Now()

	err := cs.db.DBConn.QueryRow(query, name, slug, description, now).Scan(
		&category.ID, &category.Created)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCategoryCreateFailed, err)
	}

	category.Name = name
	category.Slug = slug
	category.Description = description

	return &category, nil
}

// GetCategory получает категорию по ID
func (cs *CategoryService) GetCategory(id int) (*models.Category, error) {
	var category models.Category
	query := `SELECT id, name, slug, description, created FROM categories WHERE id = ?`

	err := cs.db.DBConn.QueryRow(query, id).Scan(
		&category.ID, &category.Name, &category.Slug,
		&category.Description, &category.Created)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	return &category, nil
}

// GetCategoryBySlug получает категорию по slug
func (cs *CategoryService) GetCategoryBySlug(slug string) (*models.Category, error) {
	var category models.Category
	query := `SELECT id, name, slug, description, created FROM categories WHERE slug = ?`

	err := cs.db.DBConn.QueryRow(query, slug).Scan(
		&category.ID, &category.Name, &category.Slug,
		&category.Description, &category.Created)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}

	return &category, nil
}

// GetAllCategories получает все категории
func (cs *CategoryService) GetAllCategories() ([]*models.Category, error) {
	query := `SELECT id, name, slug, description, created FROM categories ORDER BY name`

	rows, err := cs.db.DBConn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(&category.ID, &category.Name, &category.Slug,
			&category.Description, &category.Created)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// UpdateCategory обновляет категорию
func (cs *CategoryService) UpdateCategory(id int, name, slug, description string) error {
	if err := cs.validateCategoryData(name, slug, description); err != nil {
		return err
	}

	// Проверяем уникальность, исключая текущую категорию
	if err := cs.checkCategoryUniquenessExcluding(name, slug, id); err != nil {
		return err
	}

	query := `UPDATE categories SET name = ?, slug = ?, description = ? WHERE id = ?`
	result, err := cs.db.DBConn.Exec(query, name, slug, description, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCategoryUpdateFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

// DeleteCategory удаляет категорию
func (cs *CategoryService) DeleteCategory(id int) error {
	query := `DELETE FROM categories WHERE id = ?`
	result, err := cs.db.DBConn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCategoryDeleteFailed, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

// AssignPostToCategory назначает пост к категории
func (cs *CategoryService) AssignPostToCategory(postID, categoryID int) error {
	query := `INSERT OR IGNORE INTO post_categories (post_id, category_id) VALUES (?, ?)`
	_, err := cs.db.DBConn.Exec(query, postID, categoryID)
	return err
}

// RemovePostFromCategory удаляет пост из категории
func (cs *CategoryService) RemovePostFromCategory(postID, categoryID int) error {
	query := `DELETE FROM post_categories WHERE post_id = ? AND category_id = ?`
	_, err := cs.db.DBConn.Exec(query, postID, categoryID)
	return err
}

// GetPostCategories получает все категории поста
func (cs *CategoryService) GetPostCategories(postID int) ([]*models.Category, error) {
	query := `SELECT c.id, c.name, c.slug, c.description, c.created 
			  FROM categories c
			  JOIN post_categories pc ON c.id = pc.category_id
			  WHERE pc.post_id = ?
			  ORDER BY c.name`

	rows, err := cs.db.DBConn.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(&category.ID, &category.Name, &category.Slug,
			&category.Description, &category.Created)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetCategoryPosts получает все посты категории с пагинацией
func (cs *CategoryService) GetCategoryPosts(categoryID int, limit, offset int) ([]*models.Post, error) {
	query := `SELECT p.id, p.title, p.content, p.user_id, p.created, p.updated, u.username
			  FROM posts p
			  JOIN users u ON p.user_id = u.id
			  JOIN post_categories pc ON p.id = pc.post_id
			  WHERE pc.category_id = ?
			  ORDER BY p.created DESC
			  LIMIT ? OFFSET ?`

	rows, err := cs.db.DBConn.Query(query, categoryID, limit, offset)
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

// checkCategoryUniqueness проверяет уникальность name и slug
func (cs *CategoryService) checkCategoryUniqueness(name, slug string) error {
	var exists int

	// Проверяем name
	query := `SELECT 1 FROM categories WHERE name = ?`
	err := cs.db.DBConn.QueryRow(query, name).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrCategoryExists
		}
		return fmt.Errorf("ошибка проверки уникальности name: %v", err)
	}

	// Проверяем slug
	query = `SELECT 1 FROM categories WHERE slug = ?`
	err = cs.db.DBConn.QueryRow(query, slug).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrSlugExists
		}
		return fmt.Errorf("ошибка проверки уникальности slug: %v", err)
	}

	return nil
}

// checkCategoryUniquenessExcluding проверяет уникальность, исключая указанную категорию
func (cs *CategoryService) checkCategoryUniquenessExcluding(name, slug string, excludeID int) error {
	var exists int

	// Проверяем name
	query := `SELECT 1 FROM categories WHERE name = ? AND id != ?`
	err := cs.db.DBConn.QueryRow(query, name, excludeID).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrCategoryExists
		}
		return fmt.Errorf("ошибка проверки уникальности name: %v", err)
	}

	// Проверяем slug
	query = `SELECT 1 FROM categories WHERE slug = ? AND id != ?`
	err = cs.db.DBConn.QueryRow(query, slug, excludeID).Scan(&exists)
	if err != sql.ErrNoRows {
		if err == nil {
			return ErrSlugExists
		}
		return fmt.Errorf("ошибка проверки уникальности slug: %v", err)
	}

	return nil
}

// validateCategoryData валидирует все данные категории
func (cs *CategoryService) validateCategoryData(name, slug, description string) error {
	if err := cs.validateCategoryName(name); err != nil {
		return err
	}
	if err := cs.validateSlug(slug); err != nil {
		return err
	}
	if err := cs.validateDescription(description); err != nil {
		return err
	}
	return nil
}

// validateCategoryName валидирует название категории
func (cs *CategoryService) validateCategoryName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return ErrEmptyCategoryName
	}
	if len(name) > 100 {
		return ErrLongCategoryName
	}
	return nil
}

// validateSlug валидирует slug
func (cs *CategoryService) validateSlug(slug string) error {
	slug = strings.TrimSpace(slug)
	if len(slug) == 0 {
		return ErrEmptySlug
	}
	if len(slug) > 100 {
		return ErrLongSlug
	}

	// Проверяем формат slug (только строчные буквы, цифры и дефисы)
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, slug)
	if !matched {
		return ErrInvalidSlug
	}

	return nil
}

// validateDescription валидирует описание
func (cs *CategoryService) validateDescription(description string) error {
	if len(description) > 500 {
		return ErrLongDescription
	}
	return nil
}
