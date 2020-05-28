package tactics

// Move represents a single unambiguous chess move, including the
// square moved from, the square moved to, and the type of
// piece promoted to, if applicable.
type Move struct {
	MovedFrom       string
	MovedTo         string
	PiecePromotedTo ChessPiece
}

// NewMove creates a new move from a UCI-style string
func NewMove(uciMove string) Move {
	move := Move{
		MovedFrom: uciMove[0:2],
		MovedTo:   uciMove[2:4],
	}

	if len(uciMove) == 5 {
		promotedTo := uciMove[4]
		switch promotedTo {
		case 'q':
			fallthrough
		case 'Q':
			move.PiecePromotedTo = WhiteQueen
		case 'r':
			fallthrough
		case 'R':
			move.PiecePromotedTo = WhiteRook
		case 'b':
			fallthrough
		case 'B':
			move.PiecePromotedTo = WhiteBishop
		case 'n':
			fallthrough
		case 'N':
			move.PiecePromotedTo = WhiteKnight
		}
	}

	return move
}
