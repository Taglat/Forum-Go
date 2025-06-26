package web

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

func (app *app) ServerError(w http.ResponseWriter, err error) {
	log.Printf("%s\n%s", err.Error(), debug.Stack())
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (app *app) ClientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *app) NotFound(w http.ResponseWriter) {
	app.ClientError(w, http.StatusNotFound)
}

func (app *app) MethodNotAllowed(w http.ResponseWriter, methods []string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (app *app) Forbidden(w http.ResponseWriter) {
	app.ClientError(w, http.StatusForbidden)
}
