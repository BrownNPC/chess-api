package game

import "github.com/notnil/chess"

type Player struct {
	Username string
	Id       int
	Color    chess.Color
	Events   chan Event
}

func NewPlayer(username string, id int, color chess.Color) Player {
	return Player{
		Username: username,
		Id:       id,
		Color:    color,
		Events:   make(chan Event, 10),
	}
}
