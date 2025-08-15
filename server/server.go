package server

import (
	"api/db"
	"database/sql"
)

type Server struct {
	DB        *db.Queries
	SQL       *sql.DB
	JwtSecret []byte
}

func NewServer(dbConnection *sql.DB, jwtSecret []byte) Server {
	return Server{
		DB:        db.New(dbConnection),
		SQL:       dbConnection,
		JwtSecret: jwtSecret,
	}
}
