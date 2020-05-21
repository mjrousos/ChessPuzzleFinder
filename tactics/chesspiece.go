package tactics

import "github.com/pkg/errors"

// ChessPiece represents a combination of color and type of piece.
type ChessPiece int

// IsWhite returns true if the piece is white, false if it is black,
// and an error if the piece is invalid.
func (piece ChessPiece) IsWhite() (bool, error) {
	if piece > BlackPawn {
		return false, errors.New("Invalid chess piece")
	}

	return piece < BlackKing, nil
}

func (piece ChessPiece) String() string {
	switch piece {
	case WhiteKing:
		fallthrough
	case BlackKing:
		return "K"

	case WhiteQueen:
		fallthrough
	case BlackQueen:
		return "Q"

	case WhiteRook:
		fallthrough
	case BlackRook:
		return "R"

	case WhiteBishop:
		fallthrough
	case BlackBishop:
		return "B"

	case WhiteKnight:
		fallthrough
	case BlackKnight:
		return "N"

	case WhitePawn:
		fallthrough
	case BlackPawn:
		return ""

	default:
		return "InvalidPiece"
	}
}

const (
	// WhiteKing represents a white king.
	WhiteKing = iota

	// WhiteQueen represents a white queen.
	WhiteQueen

	// WhiteRook represents a white rook.
	WhiteRook

	// WhiteBishop represents a white bishop.
	WhiteBishop

	// WhiteKnight represents a white knight.
	WhiteKnight

	// WhitePawn represents a white pawn.
	WhitePawn

	// BlackKing represents a Black king.
	BlackKing

	// BlackQueen represents a Black queen.
	BlackQueen

	// BlackRook represents a Black rook.
	BlackRook

	// BlackBishop represents a Black bishop.
	BlackBishop

	// BlackKnight represents a Black knight.
	BlackKnight

	// BlackPawn represents a Black pawn.
	BlackPawn
)
