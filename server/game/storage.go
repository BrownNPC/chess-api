package game

import (
	"sync"
)

// map from 6 character alphanumeric game id to an ongoing game
type MatchStorage struct {
	storage map[string]*Match
	mu      sync.RWMutex
}

func NewGamesStorage() *MatchStorage {
	return &MatchStorage{
		storage: map[string]*Match{},
		mu:      sync.RWMutex{},
	}
}


// get a match, ok is false if doesnt exist
func (s *MatchStorage) GetMatch(id string) (match *Match, ok bool) {
	s.mu.RLock()
	match, ok = s.storage[id]
	s.mu.RUnlock()
	return
}
