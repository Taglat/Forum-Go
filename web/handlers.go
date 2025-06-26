package web

import (
	"forum/internal/database"
	"forum/internal/models"
	"net/http"
	"strconv"
	"strings"
)

func (app *app) home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	user := app.getCurrentUser(r)

	// Получаем посты с пагинацией (пока без неё, возьмем последние 20)
	posts, err := app.PostService.GetAllPosts(20, 0)
	if err != nil {
		app.errorLog.Printf("Failed to get posts: %v", err)
		posts = []*models.Post{} // пустой слайс при ошибке
	}

	data := &HTMLData{
		Title:       "Главная",
		Path:        r.URL.Path,
		CurrentUser: user,
		Posts:       posts,
	}

	app.RenderHTML(w, r, "home.page.html", data)
}

func (app *app) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title: "Register",
			Path:  r.URL.Path,
		}
		app.RenderHTML(w, r, "register.page.html", data)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	app.infoLog.Printf("Attempting to register user: username=%q email=%q", username, email)

	user, err := app.UserService.CreateUser(username, email, password)
	if err != nil {
		data := &HTMLData{
			Title:     "Register",
			FormError: err.Error(),
			FormData: map[string]string{
				"username": username,
				"email":    email,
			},
		}
		app.RenderHTML(w, r, "register.page.html", data)
		return
	}

	app.infoLog.Printf("Successfully registered user: %q (ID %d)", user.Username, user.ID)

	// Создаем сессию для нового пользователя
	session, err := app.SessionService.CreateSession(user.ID)
	if err != nil {
		app.errorLog.Printf("Failed to create session for user %d: %v", user.ID, err)
		// Переадресуем на login при ошибке создания сессии
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Устанавливаем cookie сессии
	app.setSessionCookie(w, session.Token)

	app.infoLog.Printf("Session created for user %q", user.Username)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *app) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title: "Login",
			Path:  r.URL.Path,
		}
		app.RenderHTML(w, r, "login.page.html", data)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	app.infoLog.Printf("Attempting to login user: email=%q", email)

	id, username, err := app.UserService.VerifyUser(email, password)
	if err != nil {
		data := &HTMLData{
			Title:     "Login",
			FormError: err.Error(),
			FormData: map[string]string{
				"email": email,
			},
		}
		app.RenderHTML(w, r, "login.page.html", data)
		return
	}

	app.infoLog.Printf("Login successful: id=%d, username=%q", id, username)

	// Создаем сессию
	session, err := app.SessionService.CreateSession(id)
	if err != nil {
		app.errorLog.Printf("Failed to create session for user %d: %v", id, err)
		app.ServerError(w, err)
		return
	}

	// Устанавливаем cookie сессии
	app.setSessionCookie(w, session.Token)

	app.infoLog.Printf("Session created for user %q", username)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *app) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.MethodNotAllowed(w, []string{"POST"})
		return
	}

	token := app.getSessionToken(r)
	if token != "" {
		if err := app.SessionService.DeleteSession(token); err != nil {
			app.errorLog.Printf("Failed to delete session: %v", err)
		}
	}

	app.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Пример приватной страницы
func (app *app) profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w, []string{"GET"})
		return
	}

	user := app.getCurrentUser(r)

	data := &HTMLData{
		Title:       "Profile",
		Path:        r.URL.Path,
		CurrentUser: user,
	}

	app.RenderHTML(w, r, "profile.page.html", data)
}

// createPost создает новый пост
func (app *app) createPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title:       "Создать пост",
			Path:        r.URL.Path,
			CurrentUser: app.getCurrentUser(r),
		}
		app.RenderHTML(w, r, "create-post.page.html", data)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	user := app.getCurrentUser(r)

	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	post, err := app.PostService.CreatePost(title, content, user.ID)
	if err != nil {
		data := &HTMLData{
			Title:       "Создать пост",
			Path:        r.URL.Path,
			FormError:   err.Error(),
			CurrentUser: user,
			FormData: map[string]string{
				"title":   title,
				"content": content,
			},
		}
		app.RenderHTML(w, r, "create-post.page.html", data)
		return
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

	if r.Method != http.MethodPost {
		data := &HTMLData{
			Title:       "Редактировать пост",
			Path:        r.URL.Path,
			CurrentUser: user,
			Post:        post,
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

	err = app.PostService.UpdatePost(id, title, content, user.ID)
	if err != nil {
		data := &HTMLData{
			Title:       "Редактировать пост",
			Path:        r.URL.Path,
			FormError:   err.Error(),
			CurrentUser: user,
			Post:        post,
			FormData: map[string]string{
				"title":   title,
				"content": content,
			},
		}
		app.RenderHTML(w, r, "edit-post.page.html", data)
		return
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
