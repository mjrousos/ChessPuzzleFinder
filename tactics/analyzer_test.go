package tactics

import "testing"

// captureGroup returns the named capture group from a regex match against s,
// or fails the test if the regex did not match or the group is missing.
func captureGroup(t *testing.T, line, group string, names []string, matches []string) string {
	t.Helper()
	if matches == nil {
		t.Fatalf("regex did not match line %q", line)
	}
	for i, n := range names {
		if n == group {
			return matches[i]
		}
	}
	t.Fatalf("regex has no named group %q (names: %v)", group, names)
	return ""
}

func TestReadyRegex(t *testing.T) {
	if !readyRegex.MatchString("readyok\n") {
		t.Error("readyRegex should match 'readyok\\n'")
	}
	if readyRegex.MatchString("readyok") {
		t.Error("readyRegex should NOT match 'readyok' without trailing newline")
	}
	if readyRegex.MatchString("notready\n") {
		t.Error("readyRegex should NOT match 'notready\\n'")
	}
}

func TestPVScoreRegex(t *testing.T) {
	line := "info depth 20 seldepth 30 multipv 1 score cp -35 nodes 12345 nps 678910 pv e2e4 e7e5"
	names := pvScoreRegex.SubexpNames()
	matches := pvScoreRegex.FindStringSubmatch(line)
	if got := captureGroup(t, line, "depth", names, matches); got != "20" {
		t.Errorf("depth = %q, want %q", got, "20")
	}
	if got := captureGroup(t, line, "pvNum", names, matches); got != "1" {
		t.Errorf("pvNum = %q, want %q", got, "1")
	}
	if got := captureGroup(t, line, "score", names, matches); got != "-35" {
		t.Errorf("score = %q, want %q", got, "-35")
	}
	if got := captureGroup(t, line, "move", names, matches); got != "e2e4" {
		t.Errorf("move = %q, want %q", got, "e2e4")
	}
}

func TestPVMateRegex(t *testing.T) {
	line := "info depth 22 seldepth 34 multipv 2 score mate 3 nodes 999 nps 1000 pv a7a8"
	names := pvMateRegex.SubexpNames()
	matches := pvMateRegex.FindStringSubmatch(line)
	if got := captureGroup(t, line, "depth", names, matches); got != "22" {
		t.Errorf("depth = %q, want %q", got, "22")
	}
	if got := captureGroup(t, line, "pvNum", names, matches); got != "2" {
		t.Errorf("pvNum = %q, want %q", got, "2")
	}
	if got := captureGroup(t, line, "score", names, matches); got != "3" {
		t.Errorf("score = %q, want %q", got, "3")
	}
	if got := captureGroup(t, line, "move", names, matches); got != "a7a8" {
		t.Errorf("move = %q, want %q", got, "a7a8")
	}
}

func TestPVScoreRegex_DoesNotMatchMateLine(t *testing.T) {
	// A mate line lacks "score cp ..." so the cp regex must not match it.
	line := "info depth 22 multipv 1 score mate 5 pv e2e4\n"
	if pvScoreRegex.MatchString(line) {
		t.Error("pvScoreRegex should NOT match a mate line")
	}
}

func TestFENRegex(t *testing.T) {
	line := "Fen: rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1\n"
	names := fenRegex.SubexpNames()
	matches := fenRegex.FindStringSubmatch(line)
	want := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	if got := captureGroup(t, line, "fen", names, matches); got != want {
		t.Errorf("fen = %q, want %q", got, want)
	}
}
