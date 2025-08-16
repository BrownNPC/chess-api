package game

import (
	"crypto/rand"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/corentings/chess"
)

type Events string
type Match struct {
	// 6 character alphanumeric game id

	StartTime, EndTime time.Time

	ID    string
	Chess *chess.Game

	// should never go above 2
	numPlayers   atomic.Uint32
	playerColors [2]chess.Color

	// forwarded along from the Moves channel
	playerEvents [2]chan Events

	sync.Mutex
	Moves chan string

	LastMove string
}

// duration is clamped between 1 minute and 12 hours.
// Increment is clamped between 1 second and 1 minute.
func (s *MatchStorage) NewMatch(duration time.Duration, increment time.Duration) *Match {
	// limit of 12 hours
	duration = max(time.Minute, duration)
	duration = min(time.Hour*12, duration)

	increment = max(time.Second, increment)
	increment = min(time.Minute, increment)
	match := Match{
		// 6 char alpha-num id
		ID:           rand.Text()[:6],
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(duration),
		Chess:        chess.NewGame(),
		numPlayers:   atomic.Uint32{},
		playerColors: [2]chess.Color{},
		playerEvents: [2]chan Events{
			make(chan Events, 10), make(chan Events, 10),
		},
		Mutex:    sync.Mutex{},
		Moves:    make(chan string),
		LastMove: "",
	}

	s.mu.Lock()
	s.storage[match.ID] = &match
	s.mu.Unlock()
	// clean up inactive match
	go func() {
		for {
			time.Sleep(time.Second * 60)
			if match.numPlayers.Load() == 0 {
				s.mu.Lock()
				delete(s.storage, match.ID)
				s.mu.Unlock()
			}
		}
	}()
	return &match
}
func (m *Match) GetPlayerCount() int {
	return int(m.numPlayers.Load())
}

// ok is false when 2 players have joined
// id is whether you're player 1 or 2
// asColor gets ignored if you aren't the first one to join.
func (m *Match) Join(asColor chess.Color) (id int, ok bool) {
	m.Lock()
	defer m.Unlock()
	if m.GetPlayerCount() > 2 {
		id := int(m.numPlayers.Add(1))
		if id == 1 {
			m.playerColors[0] = asColor
			m.playerColors[1] = asColor.Other()
		}
		return id, true
	}
	return -1, false
}

// ok is false when it's not your turn
func (m *Match) MoveAs(id int, moveAlgebraic string) (ok bool) {
	m.Lock()
	defer m.Unlock()
	if m.Chess.Position().Turn() == m.GetColor(id) {
		if err := m.Chess.MoveStr(moveAlgebraic); err != nil {
			m.playerEvents[id-1] <- Events(moveAlgebraic)
			return true
		}
	}

	return false
}

// channel that spits out events for this id
func (m *Match) Listener(id int) chan Events {
	if id < 0 || id > 2 {
		slog.Warn("Invalid player id for move in games.Match.MoveAs. Id must be between 1 or 2", "id", id)
		return make(chan Events)
	}
	return m.playerEvents[id-1]
}

// return piece color of a user id
func (m *Match) GetColor(id int) chess.Color {
	m.Lock()
	defer m.Unlock()
	if id < 0 || id > 2 {
		slog.Warn("Invalid player id for move in games.Match.MoveAs. Id must be between 1 or 2", "id", id)
		return chess.NoColor
	}
	return m.playerColors[id-1]
}
func (m *Match) Resign(id int) {
	m.Lock()
	defer m.Unlock()
	asColor := m.GetColor(id)
	if asColor == chess.NoColor {
		return
	}
	m.Chess.Resign(asColor)
}

// func (m *Match) BoardFen(id int) string {
// 	m.Lock()
// 	defer m.Unlock()
// 	m.Chess.Position().ChangeTurn()
// }
