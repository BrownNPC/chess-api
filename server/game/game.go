package game

import (
	"context"
	"crypto/rand"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/notnil/chess"
)

type EventType string

const (
	Move         EventType = "move"
	OpponentInfo           = "opponent"
	Resign                 = "resign"
)

type Event struct {
	Type EventType
	// Move in UCI notation
	Move            string `json:"move,omitempty"`
	OponentUsername string `json:"oponentUsername,omitempty"`
	// white or black
	OpponentBlack bool       `json:"opponentBlack" example:"false"`
	StartTime     *time.Time `json:"startTime,omitempty" format:"date-time"`
	EndTime       *time.Time `json:"endTime,omitempty" format:"date-time"`
}

func EventMove(opponentMove string) Event {
	return Event{
		Type: Move,
		Move: opponentMove,
	}
}
func EventResigned() Event {
	return Event{
		Type: Resign,
	}
}

// game started event is fired when the 2nd player joins.
func EventStarted(opponentUsername string, opponentBlack bool, startTime, endTime time.Time) Event {
	return Event{
		Type:            OpponentInfo,
		OponentUsername: opponentUsername,
		OpponentBlack:   opponentBlack,
		StartTime:       &startTime,
		EndTime:         &endTime,
	}
}

type Match struct {
	// 6 character alphanumeric game id

	StartTime, EndTime time.Time

	ID    string
	Chess *chess.Game

	// should never go above 2
	numPlayers atomic.Uint32
	players    [2]Player
	// delete the game
	ShutDown func()
	sync.RWMutex
}

// duration is clamped between 1 minute and 12 hours.
func (s *MatchStorage) NewMatch(duration time.Duration) *Match {
	// limit of 12 hours
	duration = max(time.Minute, duration)
	duration = min(time.Hour*12, duration)
	ctx, shutdown := context.WithCancel(context.Background())
	match := Match{
		// 6 char alpha-num id
		ID:         rand.Text()[:6],
		StartTime:  time.Now().UTC(),
		EndTime:    time.Now().UTC().Add(duration),
		Chess:      chess.NewGame(),
		numPlayers: atomic.Uint32{},
		players:    [2]Player{},
		ShutDown:   shutdown,
	}

	s.mu.Lock()
	s.storage[match.ID] = &match
	s.mu.Unlock()
	// clean up inactive match
	go func() {
		for {
			time.Sleep(time.Second * 60)
			select {
			case <-ctx.Done():
				s.mu.Lock()
				delete(s.storage, match.ID)
				s.mu.Unlock()
				return
			default:
				if match.numPlayers.Load() == 0 || time.Since(match.EndTime) > 0 {
					s.mu.Lock()
					delete(s.storage, match.ID)
					s.mu.Unlock()
					return
				}
			}
		}
	}()
	return &match
}
func (m *Match) GetPlayerCount() int {
	return int(m.numPlayers.Load())
}
func (m *Match) GetPlayerFromUsername(username string) (Player, bool) {
	m.RLock()
	defer m.RUnlock()
	for _, p := range m.players {
		if p.Username == username {
			return p, true
		}
	}
	return Player{}, false
}

// ok is false when 2 players have joined
// id is whether you're player 1 or 2
// asColor gets ignored if you aren't the first one to join.
func (m *Match) Join(username string, asColor chess.Color) (player Player, ok bool) {
	m.Lock()
	defer m.Unlock()
	if m.GetPlayerCount() < 2 {
		id := int(m.numPlayers.Add(1))
		if id == 1 {
			// player 1 gets to pick their color
			m.players[0] = NewPlayer(username, id, asColor)
			return m.players[0], true
		} else {
			// player 2 gets assined the other color
			player1 := m.players[0]
			player2 := NewPlayer(username, id, player1.Color.Other())
			m.players[1] = player2

			// broadcast EventStarted
			player1.Events <- EventStarted(player2.Username, player2.Color == chess.Black,
				m.StartTime, m.EndTime)
			// this doesn't block because channels are buffered
			player2.Events <- EventStarted(player1.Username, player1.Color == chess.Black,
				m.StartTime, m.EndTime)

			return player2, true
		}
	}
	return Player{}, false
}

// ok is false when it's not your turn
func (m *Match) MoveAs(player Player, moveStr string) bool {
	ok := m.doMove(player, moveStr)
	if !ok {
		return false
	}
	m.RLock()
	var oppEvents chan Event
	if player.Username == m.players[0].Username {
		oppEvents = m.players[1].Events
	} else {
		oppEvents = m.players[0].Events
	}
	m.RUnlock()

	// send event
	if oppEvents != nil {
		select {
		case oppEvents <- EventMove(moveStr):
		default:
			// channel full
			slog.Warn("Channel is full when trying to send event. This could be due to a slow client or something else on our side.")
		}
	}
	return true
}

func (m *Match) doMove(player Player, moveStr string) bool {
	m.Lock()
	defer m.Unlock()
	// ensure this player is in the match
	if player.Username != m.players[0].Username && player.Username != m.players[1].Username {
		return false
	}
	// check correct turn
	if m.Chess.Position().Turn() != player.Color {
		return false
	}
	// attempt move
	playedMove, err := chess.UCINotation{}.Decode(m.Chess.Position(), moveStr)
	if err != nil {
		return false
	}
	if err := m.Chess.Move(playedMove); err != nil {
		return false
	}
	return true
}

func (m *Match) Resign(player Player) {
	m.Lock()
	defer m.Unlock()
	m.Chess.Resign(player.Color)
	// close context to clean up
	defer m.ShutDown()
	var opponent Player
	if player.Id == 1 {
		opponent = m.players[1]
	} else {
		opponent = m.players[0]
	}
	opponent.Events <- EventResigned()
}

// func (m *Match) BoardFen(id int) string {
// 	m.Lock()
// 	defer m.Unlock()
// 	m.Chess.Position().ChangeTurn()
// }
