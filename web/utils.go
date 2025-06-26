package web

import (
	"forum/internal/models"
	"net/http"
)

const SessionCookieName = "session_token"

// setSessionCookie устанавливает cookie с токеном сессии
func (app *app) setSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60, // 24 часа в секундах
		HttpOnly: true,         // Защита от XSS
		Secure:   false,        // Поставить true для HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// clearSessionCookie удаляет cookie сессии
func (app *app) clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// getSessionToken получает токен сессии из cookie
func (app *app) getSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getCurrentUser получает текущего пользователя по сессии
func (app *app) getCurrentUser(r *http.Request) *models.User {
	token := app.getSessionToken(r)
	if token == "" {
		return nil
	}

	user, err := app.SessionService.GetUserBySession(token)
	if err != nil {
		return nil
	}

	return user
}

// isAuthenticated проверяет, авторизован ли пользователь
func (app *app) isAuthenticated(r *http.Request) bool {
	return app.getCurrentUser(r) != nil
}
