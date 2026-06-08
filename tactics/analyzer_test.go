package tactics

import (
	"testing"

	"github.com/spf13/viper"
)

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

func TestProcessOutputLineReady(t *testing.T) {
	state := &analysisState{}

	processOutputLine("readyok\n", state)

	if !state.ready {
		t.Error("ready should be true after readyok")
	}
}

func TestProcessOutputLineUpdatesCentipawnScores(t *testing.T) {
	viper.Set("AnalysisDepth", 99)
	t.Cleanup(func() { viper.Reset() })
	state := &analysisState{}

	processOutputLine("info depth 13 seldepth 19 multipv 1 score cp 42 nodes 123 nps 456 pv e2e4 e7e5\n", state)
	processOutputLine("info depth 12 seldepth 18 multipv 2 score cp -31 nodes 789 nps 456 pv d2d4 d7d5\n", state)

	if state.pv1Depth != 13 {
		t.Errorf("pv1Depth = %d, want %d", state.pv1Depth, 13)
	}
	if state.pv1Score != 42 {
		t.Errorf("pv1Score = %d, want %d", state.pv1Score, 42)
	}
	if state.pvMove != "e2e4" {
		t.Errorf("pvMove = %q, want %q", state.pvMove, "e2e4")
	}
	if state.pv2Depth != 12 {
		t.Errorf("pv2Depth = %d, want %d", state.pv2Depth, 12)
	}
	if state.pv2Score != -31 {
		t.Errorf("pv2Score = %d, want %d", state.pv2Score, -31)
	}
}

func TestProcessOutputLineUpdatesMateScores(t *testing.T) {
	viper.Set("AnalysisDepth", 99)
	t.Cleanup(func() { viper.Reset() })
	state := &analysisState{}

	processOutputLine("info depth 9 seldepth 15 multipv 1 score mate -2 nodes 123 nps 456 pv g7g8q\n", state)
	processOutputLine("info depth 8 seldepth 14 multipv 2 score mate 3 nodes 789 nps 456 pv a2a1q\n", state)

	if state.pv1Depth != 9 {
		t.Errorf("pv1Depth = %d, want %d", state.pv1Depth, 9)
	}
	if state.pv1Score != -10000 {
		t.Errorf("pv1Score = %d, want %d", state.pv1Score, -10000)
	}
	if state.pvMove != "g7g8" {
		t.Errorf("pvMove = %q, want %q", state.pvMove, "g7g8")
	}
	if state.pv2Depth != 8 {
		t.Errorf("pv2Depth = %d, want %d", state.pv2Depth, 8)
	}
	if state.pv2Score != 10000 {
		t.Errorf("pv2Score = %d, want %d", state.pv2Score, 10000)
	}
}

func TestProcessOutputLineTracksCurrentAndPreviousFEN(t *testing.T) {
	state := &analysisState{}
	first := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	second := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"

	processOutputLine("Fen: "+first+"\n", state)
	processOutputLine("Fen: "+second+"\n", state)

	if state.previousFen != first {
		t.Errorf("previousFen = %q, want %q", state.previousFen, first)
	}
	if state.fen != second {
		t.Errorf("fen = %q, want %q", state.fen, second)
	}
}

func TestProcessOutputLineSignalsWhenBothPrincipalVariationsReachDepth(t *testing.T) {
	viper.Set("AnalysisDepth", 10)
	t.Cleanup(func() { viper.Reset() })
	state := &analysisState{positionID: 1234, positionsAnalyzed: make(chan int, 1)}

	processOutputLine("info depth 10 seldepth 15 multipv 1 score cp 12 nodes 123 nps 456 pv e2e4 e7e5\n", state)
	select {
	case got := <-state.positionsAnalyzed:
		t.Fatalf("positionsAnalyzed received %d before pv2 reached target depth", got)
	default:
	}

	processOutputLine("info depth 10 seldepth 15 multipv 2 score cp 5 nodes 789 nps 456 pv d2d4 d7d5\n", state)
	select {
	case got := <-state.positionsAnalyzed:
		if got != 1234 {
			t.Errorf("positionsAnalyzed = %d, want %d", got, 1234)
		}
	default:
		t.Fatal("positionsAnalyzed should receive positionID after both PV depths reach target")
	}
}
