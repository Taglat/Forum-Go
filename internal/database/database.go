package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	DBConn *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	dbconn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия базы данных: %v", err)
	}

	if err := dbconn.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	database := &Database{DBConn: dbconn}

	if err := database.executeSchema(); err != nil {
		return nil, fmt.Errorf("ошибка выполнения схемы: %v", err)
	}

	log.Println("База данных успешно инициализирована")
	return database, nil
}

func (d *Database) executeSchema() error {
	schemaPath := filepath.Join("database", "../../forum.sql")
	sqlContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла схемы %s: %v", schemaPath, err)
	}

	if _, err := d.DBConn.Exec(string(sqlContent)); err != nil {
		return fmt.Errorf("ошибка выполнения схемы: %v", err)
	}

	return nil
}

func (d *Database) Close() error {
	if d.DBConn != nil {
		return d.DBConn.Close()
	}
	return nil
}

func (d *Database) Ping() error {
	return d.DBConn.Ping()
}
