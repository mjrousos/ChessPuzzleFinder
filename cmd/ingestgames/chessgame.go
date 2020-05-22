package main

import "time"

type game struct {
	// Only exported fields are encoded/decoded in JSON
	GameURL         string    `json:"GameUrl"`
	Site            int       `json:"Site"`
	Gamedate        time.Time `json:"GameDate"`
	Whiteplayerid   int       `json:"WritePlayerId"`
	Blackplayerid   int       `json:"BlackPlayerId"`
	Ucimoves        []string  `json:"UCIMoves"`
	Whiteplayername string    `json:"WhitePlayer"`
	Blackplayername string    `json:"BlackPlayer"`
}
