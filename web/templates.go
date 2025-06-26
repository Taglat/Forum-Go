package web

import (
	"bytes"
	"forum/internal/models"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
	"time"
	"unicode"
)

type HTMLData struct {
	Title       string
	Path        string
	FormError   string
	FormData    map[string]string // для хранения введённых значений в форму
	CurrentUser *models.User
	Post        *models.Post
	Posts       []*models.Post
}

var functions = template.FuncMap{
	"cap": func(str string) string {
		if str == "" {
			return ""
		}
		runes := []rune(str)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes)
	},
	"formatDate": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("02 Jan 2006, 15:04")
	},
}

func (app *app) RenderHTML(w http.ResponseWriter, r *http.Request, pageFile string, data *HTMLData) {
	if data == nil {
		data = &HTMLData{}
	}

	data.Path = r.URL.Path

	// Добавляем текущего пользователя, если он не установлен
	if data.CurrentUser == nil {
		data.CurrentUser = app.getCurrentUser(r)
	}

	layoutFile := "base.layout.html"

	files := []string{
		filepath.Join(*app.HTMLDir, layoutFile),
		filepath.Join(*app.HTMLDir, pageFile),
	}

	fm := functions

	ts, err := template.New("").Funcs(fm).ParseFiles(files...)

	if err != nil {
		log.Println(err.Error())
		app.ServerError(w, err)
		return
	}

	ts, err = ts.ParseGlob(filepath.Join(*app.HTMLDir, "*.partial.html"))
	if err != nil {
		log.Println(err.Error())
		app.ServerError(w, err)
		return
	}

	buf := new(bytes.Buffer)
	// Создаёт буфер в памяти

	err = ts.ExecuteTemplate(buf, "base", data)
	// Рендерит шаблон во временный буфер
	if err != nil {
		log.Println(err.Error())
		app.ServerError(w, err)
		return
	}

	buf.WriteTo(w)
	// Пишет рендер HTML в http.ResponseWriter, если всё прошло успешно
}
