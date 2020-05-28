package main

import "time"

type game struct {
	// Only exported fields are encoded/decoded in JSON
	GameURL            string    `json:"GameUrl"`
	Site               int       `json:"Site"`
	Gamedate           time.Time `json:"GameDate"`
	Ucimoves           []string  `json:"UCIMoves"`
	Associatedplayerid int       `json:"AssociatedPlayerId"`
	Whiteplayername    string    `json:"WhitePlayer"`
	Blackplayername    string    `json:"BlackPlayer"`
}
