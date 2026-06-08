package tactics

import "testing"

func TestNewMove(t *testing.T) {
	tests := []struct {
		name        string
		uci         string
		wantFrom    string
		wantTo      string
		wantPromote ChessPiece
	}{
		// Non-promotion cases: NewMove leaves PiecePromotedTo at the zero
		// value (the production code in data.go uses this zero-value
		// convention as the "no promotion" sentinel). Assert ChessPiece(0)
		// directly rather than coupling the test to WhiteKing's iota value.
		{"plain move", "e2e4", "e2", "e4", ChessPiece(0)},
		{"non-promoting knight move", "g1f3", "g1", "f3", ChessPiece(0)}, // UCI doesn't encode captures
		{"promote queen lower", "a7a8q", "a7", "a8", WhiteQueen},
		{"promote queen upper", "a7a8Q", "a7", "a8", WhiteQueen},
		{"promote rook lower", "h2h1r", "h2", "h1", WhiteRook},
		{"promote rook upper", "h2h1R", "h2", "h1", WhiteRook},
		{"promote bishop lower", "b7b8b", "b7", "b8", WhiteBishop},
		{"promote bishop upper", "b7b8B", "b7", "b8", WhiteBishop},
		{"promote knight lower", "c2c1n", "c2", "c1", WhiteKnight},
		{"promote knight upper", "c2c1N", "c2", "c1", WhiteKnight},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewMove(tc.uci)
			if got.MovedFrom != tc.wantFrom {
				t.Errorf("MovedFrom = %q, want %q", got.MovedFrom, tc.wantFrom)
			}
			if got.MovedTo != tc.wantTo {
				t.Errorf("MovedTo = %q, want %q", got.MovedTo, tc.wantTo)
			}
			if got.PiecePromotedTo != tc.wantPromote {
				t.Errorf("PiecePromotedTo = %v, want %v", got.PiecePromotedTo, tc.wantPromote)
			}
		})
	}
}
