package web

import (
	"forum/internal/database"
	"forum/internal/models"
	"net/http"
)

func (app *app) home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	user := app.getCurrentUser(r)

	// Получаем параметр фильтра по категории
	categorySlug := r.URL.Query().Get("category")

	var posts []*models.Post
	var err error

	if categorySlug != "" {
		// Получаем категорию по slug
		category, err := app.CategoryService.GetCategoryBySlug(categorySlug)
		if err != nil {
			if err == database.ErrCategoryNotFound {
				app.NotFound(w)
				return
			}
			app.ServerError(w, err)
			return
		}

		// Получаем посты этой категории
		posts, err = app.CategoryService.GetCategoryPosts(category.ID, 20, 0)
	} else {
		// Получаем все посты
		posts, err = app.PostService.GetAllPosts(20, 0)
	}

	if err != nil {
		app.errorLog.Printf("Failed to get posts: %v", err)
		posts = []*models.Post{}
	}

	// Для каждого поста получаем категории
	for _, post := range posts {
		categories, err := app.CategoryService.GetPostCategories(post.ID)
		if err != nil {
			app.errorLog.Printf("Failed to get categories for post %d: %v", post.ID, err)
			categories = []*models.Category{}
		}
		post.Categories = categories
	}

	// Получаем все категории для фильтра
	categories, err := app.CategoryService.GetAllCategories()
	if err != nil {
		app.errorLog.Printf("Failed to get categories: %v", err)
		categories = []*models.Category{}
	}

	data := &HTMLData{
		Title:          "Главная",
		Path:           r.URL.Path,
		CurrentUser:    user,
		Posts:          posts,
		Categories:     categories,
		FilterCategory: categorySlug,
	}

	app.RenderHTML(w, r, "home.page.html", data)
}
