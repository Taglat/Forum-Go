package web

import (
	"forum/internal/database"
	"forum/internal/models"
	"net/http"
	"strings"
)

// categories - список всех категорий
func (app *app) categories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	categories, err := app.CategoryService.GetAllCategories()
	if err != nil {
		app.ServerError(w, err)
		return
	}

	data := &HTMLData{
		Title:       "Категории",
		Path:        r.URL.Path,
		CurrentUser: app.getCurrentUser(r),
		Categories:  categories,
	}

	app.RenderHTML(w, r, "categories.page.html", data)
}

// viewCategory - просмотр постов категории
func (app *app) viewCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/category/")

	category, err := app.CategoryService.GetCategoryBySlug(slug)
	if err != nil {
		if err == database.ErrCategoryNotFound {
			app.NotFound(w)
			return
		}
		app.ServerError(w, err)
		return
	}

	posts, err := app.CategoryService.GetCategoryPosts(category.ID, 20, 0)
	if err != nil {
		app.errorLog.Printf("Failed to get category posts: %v", err)
		posts = []*models.Post{}
	}

	data := &HTMLData{
		Title:       category.Name,
		Path:        r.URL.Path,
		CurrentUser: app.getCurrentUser(r),
		Category:    category,
		Posts:       posts,
	}

	app.RenderHTML(w, r, "category.page.html", data)
}
