package web

import (
	"net/http"
)

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
