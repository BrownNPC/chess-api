package server

import (
	"github.com/corentings/chess"
	"github.com/labstack/echo/v4"
)

type MatchEvent struct {
}

// Match represents an ongoing game of chess
type Match struct {
	*chess.Game
	White, Black chan MatchEvent
}

// Authorized user can make a match and receive a sharable link for anyone to play with them
func (s Server) Matchmaking(c echo.Context) error {

	return c.JSON(200, c.Get("jwt").(*JwtClaims))
}
