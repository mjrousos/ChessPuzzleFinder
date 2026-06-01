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
		piece ChessPiece
		want  string
	}{
		{WhiteKing, "K"},
		{BlackKing, "K"},
		{WhiteQueen, "Q"},
		{BlackQueen, "Q"},
		{WhiteRook, "R"},
		{BlackRook, "R"},
		{WhiteBishop, "B"},
		{BlackBishop, "B"},
		{WhiteKnight, "N"},
		{BlackKnight, "N"},
		{WhitePawn, ""},
		{BlackPawn, ""},
		{ChessPiece(99), "InvalidPiece"},
	}
	for _, tc := range tests {
		t.Run(tc.want+"_"+tc.piece.String(), func(t *testing.T) {
			if got := tc.piece.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
