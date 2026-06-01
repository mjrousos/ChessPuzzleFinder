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
		{"plain move", "e2e4", "e2", "e4", WhiteKing},                            // WhiteKing == 0 == "no promotion" sentinel
		{"capture", "g1f3", "g1", "f3", WhiteKing},                               // 4-char moves never promote
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
