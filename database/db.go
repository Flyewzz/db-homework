package database

import (
	"github.com/jackc/pgx"
)

type DataBase struct {
	pool *pgx.ConnPool
}

var DB DataBase

func (db *DataBase) Connect() error {
	conConfig := pgx.ConnConfig{
		Host:      "127.0.0.1",
		Port:      5432,
		Database:  "forumdb",
		User:      "postgres",
		Password:  "postgres",
		TLSConfig: nil,
	}

	poolConfig := pgx.ConnPoolConfig{
		ConnConfig:     conConfig,
		MaxConnections: 35,
		AfterConnect:   nil,
		AcquireTimeout: 0,
	}

	p, err := pgx.NewConnPool(poolConfig)
	db.pool = p

	return err
}

func ErrorCode(err error) string {
	pgerr, ok := err.(pgx.PgError)
	if !ok {
		return ""
	}
	return pgerr.Code
}
