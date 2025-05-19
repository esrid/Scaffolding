package db

import (
	"database/sql"
)

type PostgresStore struct {
	DB *sql.DB
}

func NewPostgresStore(DB *sql.DB) *PostgresStore {
	return &PostgresStore{DB: DB}
}

// type Store interface {
// 	SessionStore
// 	UserStore
// 	OtpStore
// }

type AuthStore interface {
	SessionStore
	UserStore
	OtpStore
}
