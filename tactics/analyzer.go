package tactics

import "fmt"

// FindPuzzles analyzes a chess game represented by a list of UCI moves
// and returns tactical puzzles from the game (or an empty slice
// if there are no interesting tactical puzzles).
func FindPuzzles(moves []string) []Puzzle {
	fmt.Printf("Analyzing a game with %v moves\n", len(moves))
	return []Puzzle{}
}
