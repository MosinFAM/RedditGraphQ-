package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	connStr := "postgres://user:password@db:5432/postsdb?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connected to PostgreSQL successfully!")
	return db, nil
}
