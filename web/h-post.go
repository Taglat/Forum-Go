package web

import (
	"forum/internal/database"
	"forum/internal/models"
	"net/http"
	"strconv"
	"strings"
)

// createPost создает новый пост
func (app *app) createPost(w http.ResponseWriter, r *http.Request) {
	user := app.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Получаем все категории для формы
	categories, err := app.CategoryService.GetAllCategories()
	if err != nil {
		app.errorLog.Printf("Failed to get categories: %v", err)
		categories = []*models.Category{}
	}

	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title:       "Создать пост",
			Path:        r.URL.Path,
			CurrentUser: user,
			Categories:  categories,
		}
		app.RenderHTML(w, r, "create-post.page.html", data)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))

	// Получаем выбранные категории
	selectedCategories := r.Form["categories"]

	var categoryIDs []int
	for _, categoryIDStr := range selectedCategories {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			continue
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	// Передаем categoryIDs в CreatePost
	post, err := app.PostService.CreatePost(title, content, user.ID, categoryIDs)
	if err != nil {
		data := &HTMLData{
			Title:       "Создать пост",
			Path:        r.URL.Path,
			FormError:   err.Error(),
			CurrentUser: user,
			Categories:  categories,
			FormData: map[string]string{
				"title":   title,
				"content": content,
			},
		}
		app.RenderHTML(w, r, "create-post.page.html", data)
		return
	}

	// Привязываем пост к выбранным категориям
	for _, categoryIDStr := range selectedCategories {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			continue
		}

		if err := app.CategoryService.AssignPostToCategory(post.ID, categoryID); err != nil {
			app.errorLog.Printf("Failed to assign post %d to category %d: %v",
				post.ID, categoryID, err)
		}
	}

	app.infoLog.Printf("Post created: ID=%d, Title=%q, Author=%q",
		post.ID, post.Title, user.Username)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// viewPost показывает отдельный пост
func (app *app) viewPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/post/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.NotFound(w)
		return
	}

	post, err := app.PostService.GetPost(id)
	if err != nil {
		if err == database.ErrPostNotFound {
			app.NotFound(w)
			return
		}
		app.ServerError(w, err)
		return
	}

	data := &HTMLData{
		Title:       post.Title,
		Path:        r.URL.Path,
		CurrentUser: app.getCurrentUser(r),
		Post:        post,
	}

	app.RenderHTML(w, r, "view-post.page.html", data)
}

// editPost редактирует пост
func (app *app) editPost(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/post/")
	idStr = strings.TrimSuffix(idStr, "/edit")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.NotFound(w)
		return
	}

	user := app.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	post, err := app.PostService.GetPost(id)
	if err != nil {
		app.NotFound(w)
		return
	}

	// Проверяем, что пользователь - автор поста
	if post.UserID != user.ID {
		app.Forbidden(w)
		return
	}

	// Получаем все категории
	allCategories, err := app.CategoryService.GetAllCategories()
	if err != nil {
		app.errorLog.Printf("Failed to get categories: %v", err)
		allCategories = []*models.Category{}
	}

	// Получаем категории поста
	postCategories, err := app.CategoryService.GetPostCategories(id)
	if err != nil {
		app.errorLog.Printf("Failed to get post categories: %v", err)
		postCategories = []*models.Category{}
	}

	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title:          "Редактировать пост",
			Path:           r.URL.Path,
			CurrentUser:    user,
			Post:           post,
			Categories:     allCategories,
			PostCategories: postCategories,
			FormData: map[string]string{
				"title":   post.Title,
				"content": post.Content,
			},
		}
		app.RenderHTML(w, r, "edit-post.page.html", data)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	selectedCategories := r.Form["categories"]

	var categoryIDs []int
	for _, categoryIDStr := range selectedCategories {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			continue
		}
		categoryIDs = append(categoryIDs, categoryID)
	}

	err = app.PostService.UpdatePost(id, title, content, categoryIDs, user.ID)
	if err != nil {
		data := &HTMLData{
			Title:          "Редактировать пост",
			Path:           r.URL.Path,
			FormError:      err.Error(),
			CurrentUser:    user,
			Post:           post,
			Categories:     allCategories,
			PostCategories: postCategories,
			FormData: map[string]string{
				"title":   title,
				"content": content,
			},
		}
		app.RenderHTML(w, r, "edit-post.page.html", data)
		return
	}

	// Удаляем все старые связи с категориями
	for _, category := range postCategories {
		app.CategoryService.RemovePostFromCategory(id, category.ID)
	}

	// Добавляем новые связи
	for _, categoryIDStr := range selectedCategories {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			continue
		}

		if err := app.CategoryService.AssignPostToCategory(id, categoryID); err != nil {
			app.errorLog.Printf("Failed to assign post %d to category %d: %v",
				id, categoryID, err)
		}
	}

	app.infoLog.Printf("Post updated: ID=%d, Title=%q, Author=%q",
		id, title, user.Username)

	http.Redirect(w, r, "/post/"+strconv.Itoa(id), http.StatusSeeOther)
}

// deletePost удаляет пост
func (app *app) deletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.MethodNotAllowed(w, []string{"POST"})
		return
	}

	idStr := r.FormValue("post_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.NotFound(w)
		return
	}

	user := app.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err = app.PostService.DeletePost(id, user.ID)
	if err != nil {
		app.errorLog.Printf("Failed to delete post %d: %v", id, err)
		app.ServerError(w, err)
		return
	}

	app.infoLog.Printf("Post deleted: ID=%d, Author=%q", id, user.Username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
