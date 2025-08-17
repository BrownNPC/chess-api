// handlers for creating and joining matches
package server

import (
	"api/server/game"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/notnil/chess"
	"github.com/notnil/chess/image"
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
	Match := s.GameStorage.NewMatch(req.Duration * time.Hour)
	return c.JSON(200, MatchCreatedResponse{Match.ID})
}

type CreateMatchRequest struct {
	// duration in hours
	// max is 12 hours
	Duration time.Duration `json:"duration" example:"120"`
}

type JoinMatchRequest struct {
	// whether to use black pieces instead of white
	BlackPieces bool `json:"blackPieces" example:"false"`
}

func (s Server) JoinMatch(c echo.Context) error {
	username := c.Get("username").(string)
	if username == "" {
		return c.JSON(http.StatusForbidden, REASON_UNAUTHORIZED)
	}

	matchID := c.Param("id")
	match, ok := s.GameStorage.GetMatch(matchID)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("Match not found"))
	}

	var req JoinMatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}

	// SSE headers
	// Ensure the writer supports Flush

	w := c.Response()
	w.Header().Set(echo.HeaderContentType, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	w.Flush()

	var asColor chess.Color
	if req.BlackPieces {
		asColor = chess.Black
	} else {
		asColor = chess.White
	}

	player, ok := match.Join(username, asColor)
	if !ok {
		return c.JSON(http.StatusForbidden, Reason("Match is full"))
	}

	// Ensure the player is removed when this handler returns (disconnect, error, etc.)
	defer match.Resign(player)

	// ticker for keep-alive
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	ctx := c.Request().Context()

	var b strings.Builder

	for {
		select {
		case <-ctx.Done():
			// client disconnected
			return nil

		case <-ticker.C:
			// send a comment keep-alive line (SSE comment)
			if _, err := w.Write([]byte(": keep-alive\n\n")); err != nil {
				return nil
			}
			w.Flush()

		case e := <-player.Events:
			msg, err := json.Marshal(e)
			if err != nil {
				// don't break loop — log and continue
				slog.Warn("Failed to marshal match.Event", "error", err)
				continue
			}

			b.WriteString("data: ")
			b.Write(msg)
			b.WriteString("\n\n")

			if _, err := w.Write([]byte(b.String())); err != nil {
				return nil
			}
			w.Flush()
			b.Reset()
			if e.Type == game.Resign {
				return nil
			}
		}
	}
}

type PutMoveRequest struct {
	Move string `json:"move" example:"e2e4"`
}

func (s Server) PutMove(c echo.Context) error {
	username := c.Get("username").(string)
	matchId := c.Param("id")

	if username == "" {
		return c.JSON(http.StatusForbidden, REASON_UNAUTHORIZED)
	}

	var req PutMoveRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}

	Match, ok := s.GameStorage.GetMatch(matchId)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("match not found"))
	}

	plr, ok := Match.GetPlayerFromUsername(username)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("Player not in-game"))
	}

	ok = Match.MoveAs(plr, req.Move)
	if !ok {
		return c.JSON(http.StatusBadRequest, Reason("Invalid move"))
	}
	return c.JSON(http.StatusOK, "ok")
}
func (s Server) GetBoardString(c echo.Context) error {
	matchId := c.Param("id")

	Match, ok := s.GameStorage.GetMatch(matchId)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("match not found"))
	}

	Match.RLock()
	defer Match.RUnlock()
	var position string = Match.Chess.Position().Board().Draw()
	return c.String(http.StatusOK, position)
}
func (s Server) GetBoardImage(c echo.Context) error {
	username := c.Get("username").(string)
	if username == "" {
		return c.JSON(http.StatusForbidden, REASON_UNAUTHORIZED)
	}
	matchId := c.Param("id")

	Match, ok := s.GameStorage.GetMatch(matchId)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("match not found"))
	}

	Match.RLock()
	defer Match.RUnlock()
	var position = Match.Chess.Position().Board()

	c.Response().Header().Set(echo.HeaderContentType, "image/svg+xml")
	c.Response().WriteHeader(http.StatusOK)

	// pass the response writer to your function
	if err := image.SVG(c.Response().Writer, position); err != nil {
		return err
	}
	return nil
}
