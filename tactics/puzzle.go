package tactics

// Puzzle represents a one-move chess tactics puzzle.
type Puzzle struct {
	Position      string
	SetupMove     Move
	CorrectMove   Move
	IncorrectMove Move
}
