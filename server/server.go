package server

import (
	"api/db"
	"api/server/game"
	"database/sql"
)

type Server struct {
	DB          *db.Queries
	SQL         *sql.DB
	JwtSecret   []byte
	GameStorage *game.MatchStorage
}

func NewServer(dbConnection *sql.DB, jwtSecret []byte) Server {
	return Server{
		DB:          db.New(dbConnection),
		SQL:         dbConnection,
		JwtSecret:   jwtSecret,
		GameStorage: game.NewGamesStorage(),
	}
}
