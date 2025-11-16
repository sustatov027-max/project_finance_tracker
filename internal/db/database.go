package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() error {
	connStr := "user=postgres password=2556625 dbname=finance_tracker sslmode=disable client_encoding=UTF8"

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	return DB.Ping()
}
