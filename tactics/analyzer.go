package tactics

import (
	"context"
	"log"
	"time"
)

// FindPuzzles analyzes a chess game represented by a list of UCI moves
// and returns tactical puzzles from the game (or an empty slice
// if there are no interesting tactical puzzles).
func FindPuzzles(ctx context.Context, moves []string) []Puzzle {
	log.Printf("Analyzing a game with %d moves\n", len(moves))

	// TODO
	time.Sleep(20 * time.Second)

	return []Puzzle{}
}
