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

// Authorized users can make a match and receive a sharable link for anyone to play with them.
// @Summary Create a match, and get a sharable link.
// @Description Authorized users can make a match and receive a sharable link for anyone to play with them.
// @Tags games
// @Accept json
// @Return json
// @Param Authorization header string true "Must contain JWT from auth/login in the format Bearer: <JWT>"
// @Router /games [post]
func (s Server) Matchmaking(c echo.Context) error {
	return c.JSON(200, c.Get("username"))
}
