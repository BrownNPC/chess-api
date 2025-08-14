package server

import (
	"api/db"
	"database/sql"
)

type Server struct {
	DB  *db.Queries
	SQL *sql.DB
}

func NewServer(dbConnection *sql.DB) Server {
	return Server{
		DB:  db.New(dbConnection),
		SQL: dbConnection,
	}
}
