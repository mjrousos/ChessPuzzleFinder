package tactics

import "testing"

func TestChessPiece_IsWhite(t *testing.T) {
	tests := []struct {
		name      string
		piece     ChessPiece
		wantWhite bool
		wantErr   bool
	}{
		{"WhiteKing", WhiteKing, true, false},
		{"WhiteQueen", WhiteQueen, true, false},
		{"WhiteRook", WhiteRook, true, false},
		{"WhiteBishop", WhiteBishop, true, false},
		{"WhiteKnight", WhiteKnight, true, false},
		{"WhitePawn", WhitePawn, true, false},
		{"BlackKing", BlackKing, false, false},
		{"BlackQueen", BlackQueen, false, false},
		{"BlackRook", BlackRook, false, false},
		{"BlackBishop", BlackBishop, false, false},
		{"BlackKnight", BlackKnight, false, false},
		{"BlackPawn", BlackPawn, false, false},
		{"AboveBlackPawn", ChessPiece(BlackPawn + 1), false, true},
		{"FarOutOfRange", ChessPiece(99), false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotWhite, err := tc.piece.IsWhite()
			if (err != nil) != tc.wantErr {
				t.Fatalf("IsWhite() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if err != nil && err.Error() != "Invalid chess piece" {
				t.Errorf("IsWhite() error message = %q, want %q", err.Error(), "Invalid chess piece")
			}
			if gotWhite != tc.wantWhite {
				t.Errorf("IsWhite() = %v, want %v", gotWhite, tc.wantWhite)
			}
		})
	}
}

func TestChessPiece_String(t *testing.T) {
	tests := []struct {
		name  string
		piece ChessPiece
		want  string
	}{
		{"WhiteKing", WhiteKing, "K"},
		{"BlackKing", BlackKing, "K"},
		{"WhiteQueen", WhiteQueen, "Q"},
		{"BlackQueen", BlackQueen, "Q"},
		{"WhiteRook", WhiteRook, "R"},
		{"BlackRook", BlackRook, "R"},
		{"WhiteBishop", WhiteBishop, "B"},
		{"BlackBishop", BlackBishop, "B"},
		{"WhiteKnight", WhiteKnight, "N"},
		{"BlackKnight", BlackKnight, "N"},
		{"WhitePawn", WhitePawn, ""},
		{"BlackPawn", BlackPawn, ""},
		{"OutOfRange", ChessPiece(99), "InvalidPiece"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.piece.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
