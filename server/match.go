// handlers for creating and joining matches
package server

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// MatchCreatedResponse is the information needed to join a match as the owner or as the opponent
type MatchCreatedResponse struct {
	ID string `json:"matchId" example:"AB2C21"`
}

// Authorized users can make a match and receive a game id, which other people can use to join the match.
// @Summary Create a match, and get a sharable match id.
// @Description Authorized users can make a match and receive a game id, which other users can use to join the match.
// @Description You must be the first one to use the id at /matches/:id if you want to be the one who picks the colors.
// @Description Duration maxes out at 43200 (12 hours) and increment maxes out at 60.
// @Description field "black":bool is whether to join as the black pieces or white pieces
// @Tags matches
// @Accept json
// @Return json
// @Param Authorization header string true "Must contain ApiKey in the format Bearer: <apiKey>"
// @Router /matches [post]
func (s Server) CreateMatch(c echo.Context) error {
	username := c.Get("username").(string)
	if username == "" {
		return c.JSON(http.StatusForbidden, Reason("You need to be authorized to make a match"))
	}
	var req CreateMatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}
	match := s.GameStorage.NewMatch(req.Duration*time.Second,
		req.Increment*time.Second)
	return c.JSON(200, MatchCreatedResponse{match.ID})
}

type CreateMatchRequest struct {
	// duration in seconds
	// max is 43200 (12 hours)
	Duration time.Duration `json:"duration" example:"120"`
	// increment in seconds
	// max is 60
	Increment time.Duration `json:"increment"`
}

type JoinMatchRequest struct {
	// can be "black" or "white"
	Color string `json:"color" example:"black" example:"white"`
}

func (s Server) JoinMatch(c echo.Context) error {
	matchId := c.Param("id")
	match, ok := s.GameStorage.GetMatch(matchId)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("Match not found"))
	}

	var req JoinMatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}

	w := c.Response()
	w.Header().Set(echo.HeaderContentType, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.WriteHeader(http.StatusOK)
	w.Flush()

	id, ok := match.Join()
	if !ok {
		return c.JSON(http.StatusForbidden, Reason("Match is full"))
	}

	for e := range match.Listener(id) {

	}

	return nil
}
