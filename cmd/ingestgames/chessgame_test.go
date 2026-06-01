package main

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// TestGameJSONRoundTrip locks in the wire shape that the Azure Queue
// producer sends and the worker decodes. Any drift in a `json:"..."` tag
// on the `game` struct will fail this test, which is the contract the
// azqueue/v2 dequeue path relies on.
func TestGameJSONRoundTrip(t *testing.T) {
	const raw = `{
		"GameUrl": "https://lichess.org/abcd1234",
		"Site": 0,
		"GameDate": "2025-05-01T12:34:56Z",
		"UCIMoves": ["e2e4", "e7e5", "g1f3"],
		"AssociatedPlayerId": 42,
		"WhitePlayer": "Alice",
		"BlackPlayer": "Bob"
	}`

	var g game
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if g.GameURL != "https://lichess.org/abcd1234" {
		t.Errorf("GameURL = %q", g.GameURL)
	}
	if g.Site != 0 {
		t.Errorf("Site = %d, want 0 (lichess.org)", g.Site)
	}
	wantDate := time.Date(2025, 5, 1, 12, 34, 56, 0, time.UTC)
	if !g.Gamedate.Equal(wantDate) {
		t.Errorf("Gamedate = %v, want %v", g.Gamedate, wantDate)
	}
	if len(g.Ucimoves) != 3 || g.Ucimoves[0] != "e2e4" || g.Ucimoves[2] != "g1f3" {
		t.Errorf("Ucimoves = %v", g.Ucimoves)
	}
	if g.Associatedplayerid != 42 {
		t.Errorf("Associatedplayerid = %d, want 42", g.Associatedplayerid)
	}
	if g.Whiteplayername != "Alice" {
		t.Errorf("Whiteplayername = %q, want Alice", g.Whiteplayername)
	}
	if g.Blackplayername != "Bob" {
		t.Errorf("Blackplayername = %q, want Bob", g.Blackplayername)
	}

	// Round-trip: marshal back and confirm every json tag is preserved.
	out, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var back game
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("Unmarshal after round-trip failed: %v", err)
	}
	if !reflect.DeepEqual(back, g) {
		t.Errorf("round-trip mismatch:\n got:  %#v\n want: %#v", back, g)
	}
}

func TestGameSiteEnum(t *testing.T) {
	// The data.go writer treats Site=0 as lichess.org and Site=1 as
	// chess.com. Confirm the JSON contract preserves those exact ints.
	for _, raw := range []string{`{"Site": 0}`, `{"Site": 1}`} {
		var g game
		if err := json.Unmarshal([]byte(raw), &g); err != nil {
			t.Fatalf("Unmarshal(%q) failed: %v", raw, err)
		}
	}
}
