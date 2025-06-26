package web

import (
	"net/http"
	"regexp"
)

func (app *app) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir(*app.StaticDir))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/", app.home)

	// Маршруты только для гостей (неавторизованных)
	mux.HandleFunc("/register", app.requireGuest(app.register))
	mux.HandleFunc("/login", app.requireGuest(app.login))

	// Маршруты только для авторизованных пользователей
	mux.HandleFunc("/logout", app.requireAuth(app.logout))
	mux.HandleFunc("/profile", app.requireAuth(app.profile))

	mux.HandleFunc("/post/create", app.requireAuth(app.createPost))
	mux.HandleFunc("/post/delete", app.requireAuth(app.deletePost))
	mux.HandleFunc("/post/", app.handlePostRoutes)

	return mux
}

// handlePostRoutes обрабатывает динамические маршруты постов
func (app *app) handlePostRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /post/{id}
	if matches := regexp.MustCompile(`^/post/(\d+)$`).FindStringSubmatch(path); matches != nil {
		app.viewPost(w, r)
		return
	}

	// /post/{id}/edit
	if matches := regexp.MustCompile(`^/post/(\d+)/edit$`).FindStringSubmatch(path); matches != nil {
		app.editPost(w, r)
		return
	}

	app.NotFound(w)
}

// | Что                | Тип         | Назначение                                | Пример использования                          |
// | ------------------ | ----------- | ----------------------------------------- | --------------------------------------------- |
// | `http.Handle`      | функция     | Назначает `http.Handler` на путь          | http.Handle("/home", myHandler)               |
// |                    |             |                                           | // myHandler должен иметь метод ServeHTTP     |
// | `http.HandleFunc`  | функция     | Назначает `func(w, r)` на путь            | http.HandleFunc("/about", handlerFunc)        |
// |                    |             |                                           | // Принимает обычную функцию как аргумент     |
// | `http.Handler`     | интерфейс   | Абстракция: что-то, что умеет `ServeHTTP` | type MyHandler struct{}                       |
// |                    |             |                                           | func (h MyHandler) ServeHTTP(w, r) { ... }    |
// |                    |             |                                           | // Реализация интерфейса                      |
// | `http.HandlerFunc` | тип-адаптер | Превращает функцию в `http.Handler`       | http.Handle("/x", http.HandlerFunc(fn))       |
// |                    |             |                                           | // fn: func(w http.ResponseWriter, r *http.Request) |

// 1. http.Handle(pattern string, handler http.Handler)
// Это функция, которая регистрирует обработчик (http.Handler) для указанного пути (маршрута).
//
// Пример:
//     http.Handle("/home", myHandler)
//
// Здесь myHandler должен реализовать интерфейс http.Handler:
//     type Handler interface {
//         ServeHTTP(ResponseWriter, *Request)
//     }
//
// То есть:
//     type MyHandler struct{}
//     func (h MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//         fmt.Fprint(w, "Hello")
//     }
//
//     http.Handle("/home", MyHandler{})

// 2. http.HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
// Это функция, которая регистрирует обычную функцию как обработчик на указанный путь.
//
// Пример:
//     http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
//         fmt.Fprint(w, "About page")
//     })
//
// Под капотом оборачивает функцию в http.HandlerFunc, реализующий интерфейс http.Handler.

// 3. http.Handler (интерфейс)
// Это базовый интерфейс, который должен реализовать любой HTTP-обработчик в Go.
//
// Интерфейс:
//     type Handler interface {
//         ServeHTTP(ResponseWriter, *Request)
//     }
//
// Всё, что реализует этот интерфейс, можно использовать как обработчик в http.Server или http.Handle.

// 4. http.HandlerFunc (тип-адаптер)
// Это тип-обёртка, позволяющая обычной функции с сигнатурой
//     func(http.ResponseWriter, *http.Request)
// реализовать интерфейс http.Handler.
//
// Тип:
//     type HandlerFunc func(ResponseWriter, *Request)
//
// Реализация:
//     func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
//         f(w, r)
//     }
//
// Пример:
//     myFunc := func(w http.ResponseWriter, r *http.Request) {
//         fmt.Fprint(w, "Hello from HandlerFunc")
//     }
//     http.Handle("/x", http.HandlerFunc(myFunc))
