package server

import (
	"api/db"
	"database/sql"
)

type Server struct {
	DB  *db.Queries
	SQL *sql.DB
}

// Error reason
type ErrorReason struct {
	Reason string `json:"reason" example:"<reason for failure>"`
}

func Reason(err string) ErrorReason {
	return ErrorReason{err}
}

func NewServer(dbConnection *sql.DB) Server {
	return Server{
		DB:  db.New(dbConnection),
		SQL: dbConnection,
	}
}
