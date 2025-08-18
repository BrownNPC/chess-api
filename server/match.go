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
//
//	@Summary		Create a match, and get a sharable match id.
//	@Description	**Authorized users** can make a match and receive a game id, which other users can use to join the match.
//	@Description	### Note:
//	@Description	### You must be the first one to send a GET to /matches/:id if you want to be the one who picks the colors.
//	@Description	### duration maxes out at 12 hours
//	@Tags			matches
//	@Param			Authorization	header	string				true	"Must contain ApiKey in the format Bearer: apiKey"
//	@Param			payload			body	CreateMatchRequest	true	"Duration of the match in hours. Max is 12"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	MatchCreatedResponse	"Match Created"
//	@Failure		403	{object}	ErrorReason				"Invalid Authorization header"
//	@Failure		400	{object}	ErrorReason				"Invalid json body"
//	@Router			/matches [post]
func (s Server) CreateMatch(c echo.Context) error {
	username := c.Get("username").(string)
	if username == "" {
		return c.JSON(http.StatusForbidden, Reason("You need to be authorized to make a match"))
	}
	var req CreateMatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, REASON_JSON_SYNTAX_ERROR)
	}
	if req.Duration == 0 {
		return c.JSON(http.StatusBadRequest, Reason("Duration not provided"))
	}
	Match := s.GameStorage.NewMatch(time.Duration(req.Duration) * time.Hour)
	return c.JSON(200, MatchCreatedResponse{Match.ID})
}

type CreateMatchRequest struct {
	Duration int `json:"duration" example:"12"` // duration in hours
}

type JoinMatchRequest struct {
	// whether to use black pieces instead of white
	BlackPieces bool `json:"blackPieces" example:"false"`
}

// Authorized users can join an existing match using a game id.
//
//	@Summary		Join a match and receive events from the server.
//	@Description	Authorized users can join a match using the game id.
//	@Description	The first person to join choeses their color.
//	@Description	## On success the server will send `SSE` messages whose payloads are JSON.
//	@Description	Events don't send this entire object: each event uses only some fields.
//	@Description	Look [here](https://github.com/BrownNPC/chess-api/blob/master/server/game/game.go#L33) to see **which fields are used by which event.**
//	@Tags			matches
//	@Accept			json
//	@Produce		json
//	@Produce		event-stream
//	@Param			Authorization	header		string				true	"Must contain ApiKey in the format Bearer: apiKey"
//	@Param			id				path		string				true	"Match ID"
//	@Param			payload			body		JoinMatchRequest	true	"`blackPieces` is used to pick if you want to play as the black pieces. This is ignored if you are not the first one to join."
//	@Success		200				{object}	game.Event			"SSE stream — each `data:` payload uses some fields of this JSON object (Content-Type: text/event-stream). Events dont sent this whole object."
//	@Failure		403				{object}	ErrorReason			"Unauthorized"
//	@Failure		404				{object}	ErrorReason			"Match not found"
//	@Failure		400				{object}	ErrorReason			"Invalid json body"
//	@Router			/matches/{id}/play [get]
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

// @Summary		players in-game can make moves when it's their turn.
// @Description	You must be in-game to post a move.
// @Description	The move needs to be in UCI format. eg. `e2e4`
// @Description	You cannot make a move if it's not your turn.
// @Param			Authorization	header	string			true	"Must contain ApiKey in the format Bearer: apiKey"
// @Param			payload			body	PutMoveRequest	true	"move in UCI notation. eg. e2e4"
// @Param			id				path	string			true	"Match ID"
// @Tags			matches
// @Accept			json
// @Produce		json
// @Failure		403	{object}	ErrorReason	"Unauthorized"
// @Failure		404	{object}	ErrorReason	"Match not found"
// @Failure		400	{object}	ErrorReason	"Invalid json body / invalid move"
// @Success		200	{object}	string		"ok"
// @Router			/matches/{id}  [put]
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

// @Summary		Get board in FEN format.
// @Description	Get the board position in FEN format.
// @Description	Unauthorized clients can use this.
// @Tags			matches
// @Accept			json
// @Produce		json
// @Failure		404	{object}	ErrorReason	"Match not found"
// @Failure		400	{object}	ErrorReason	"Invalid json body / invalid move"
// @Success		200	{object}	string		"board FEN"
// @Param			id	path		string		true	"Match ID"
// @Router			/matches/{id}  [get]
func (s Server) GetBoardFEN(c echo.Context) error {
	matchId := c.Param("id")

	Match, ok := s.GameStorage.GetMatch(matchId)
	if !ok {
		return c.JSON(http.StatusNotFound, Reason("match not found"))
	}

	Match.RLock()
	defer Match.RUnlock()
	var position string = Match.Chess.Position().Board().String()
	return c.String(http.StatusOK, position)
}

// @Summary		Get board in SVG format.
// @Description	Get the board position in SVG Image format.
// @Tags			matches
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string		true	"Must contain ApiKey in the format Bearer: apiKey"
// @Param			id				path		string		true	"Match ID"
// @Failure		403				{object}	ErrorReason	"Unauthorized"
// @Failure		404				{object}	ErrorReason	"Match not found"
// @Failure		400				{object}	ErrorReason	"Invalid json body / invalid move"
// @Success		200				{file}		string		"SVG image"
// @Router			/matches/{id}/img  [get]
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
