package web

import (
	"database/sql"
	"flag"
	"forum/internal/database"
	"log"
	"net/http"
	"os"
	"time"
)

type app struct {
	infoLog        *log.Logger
	errorLog       *log.Logger
	HTMLDir        *string
	StaticDir      *string
	Database       *database.Database
	UserService    *database.UserService
	SessionService *database.SessionService
	PostService    *database.PostService
}

func RunApp() {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	addr := flag.String("addr", ":4000", "HTTP network address")
	htmlDir := flag.String("html-dir", "./ui/html", "Path to HTML templates")
	staticDir := flag.String("static-dir", "./ui/static", "Path to static assets")
	dsn := flag.String("dsn", "./forum.db", "Path to SQLite3 database file")

	flag.Parse()

	dbConn, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		errorLog.Fatal("Failed to open SQLite DB:", err)
	}

	if err := dbConn.Ping(); err != nil {
		errorLog.Fatal("Failed to ping SQLite DB:", err)
	}

	defer dbConn.Close()

	infoLog.Println("SQLite DB connected:", *dsn)

	db := &database.Database{DBConn: dbConn}
	userService := database.NewUserService(db)
	sessionService := database.NewSessionService(db)
	postService := database.NewPostService(db)

	app := &app{
		errorLog:       errorLog,
		infoLog:        infoLog,
		HTMLDir:        htmlDir,
		StaticDir:      staticDir,
		Database:       &database.Database{DBConn: dbConn},
		UserService:    userService,
		SessionService: sessionService,
		PostService:    postService,
	}

	if err := app.SessionService.CleanupExpiredSessions(); err != nil {
		app.infoLog.Printf("Warning: failed to cleanup expired sessions: %v", err)
	}

	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: app.errorLog,
		Handler:  app.routes(),

		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on http://localhost%s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		errorLog.Fatal(err)
	}
}
