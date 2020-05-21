package tactics

// Move represents a single unambiguous chess move, including the type of
// piece moved, the square moved from, the square moved to, and the type of
// piece promoted to, if applicable.
type Move struct {
	PieceMoved      ChessPiece
	MovedFrom       string
	MovedTo         string
	PiecePromotedTo ChessPiece
}
