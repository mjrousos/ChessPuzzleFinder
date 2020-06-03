package tactics

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var readyRegex = regexp.MustCompile(`^readyok\n$`)
var pvScoreRegex = regexp.MustCompile(`^info depth (?P<depth>\d+) .*multipv (?P<pvNum>\d+).*score cp (?P<score>-?\d+).*pv (?P<move>[a-h1-8]+)`)
var pvMateRegex = regexp.MustCompile(`^info depth (?P<depth>\d+) .*multipv (?P<pvNum>\d+).*score mate (?P<score>-?\d+).*pv (?P<move>[a-h1-8]+)`)
var fenRegex = regexp.MustCompile(`^Fen: (?P<fen>.*)\n$`)

type analysisState struct {
	ready             bool
	pvMove            string
	pv1Score          int
	pv1Depth          int
	pv2Score          int
	pv2Depth          int
	fen               string
	previousFen       string
	puzzles           []Puzzle
	positionID        int
	positionsAnalyzed chan int
}

func (state *analysisState) nextPosition() {
	state.positionID = rand.Int()
	state.pv1Depth = 0
	state.pv2Depth = 0
}

// FindPuzzles analyzes a chess game represented by a list of UCI moves
// and returns tactical puzzles from the game (or an empty slice
// if there are no interesting tactical puzzles).
func FindPuzzles(ctx context.Context, moves []string) []Puzzle {
	log.Printf("Analyzing a game with %d moves\n", len(moves))

	enginePath := viper.GetString("EnginePath")
	cmd := exec.Command(enginePath)
	log.Printf("Starting %s\n", enginePath)
	engineOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to pipe engine output: %v\n", err)
	}
	engineIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to pipe engine input: %v\n", err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start engine: %v\n", err)
	}

	state := analysisState{puzzles: make([]Puzzle, 0), positionsAnalyzed: make(chan int, 10)}
	done := make(chan int)
	go processOutput(engineOut, &state, done)
	initializeEngineSession(engineIn)

	// The first several moves are unlikely to reveal an interesting tactic, so start iterator at 6
	for i := 6; i < len(moves); i++ {
		select {
		case <-ctx.Done():
			// If the app is interrupted, cancel analysis
			break
		default:
			analyzeMove(engineIn, moves[:i], moves[i], &state)
		}
	}

	// Close engine
	engineIn.Write([]byte("quit"))
	engineIn.Close()
	cmd.Wait()
	<-done

	return state.puzzles
}

func analyzeMove(input io.WriteCloser, moves []string, nextMove string, state *analysisState) {
	// Set position
	setPositionCommand := fmt.Sprintf("position startpos moves %v\n", strings.Join(moves, " "))
	state.nextPosition()
	input.Write([]byte(setPositionCommand))

	// Analyze
	input.Write([]byte("go infinite\n"))

	// Wait for analysis
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(viper.GetInt("AnalysisSecondsPerMove")) * time.Second)
		timeout <- true
	}()
	for analysisDone := false; !analysisDone; {
		select {
		case <-timeout:
			analysisDone = true
		case puzzleEvaluated := <-state.positionsAnalyzed:
			if puzzleEvaluated == state.positionID {
				analysisDone = true
			}
		}
	}

	// Stop analysis, get FEN, and wait to make sure stdout has been written to
	input.Write([]byte("stop\n"))
	input.Write([]byte("d\n"))
	time.Sleep(100 * time.Millisecond)
	log.Printf("Evaluated: %s to depth %d\n", state.fen, min(state.pv1Depth, state.pv2Depth))

	if nextMove != state.pvMove && // If the player didn't make the best move
		abs(state.pv1Score-state.pv2Score) >= 300 && // and not making the best move was a blunder
		(abs(state.pv1Score) <= 600 || abs(state.pv2Score) <= 600) { // and the game wasn't already essentially decided
		newpuzzle := Puzzle{
			Position:      state.previousFen,
			SetupMove:     NewMove(moves[len(moves)-1]),
			CorrectMove:   NewMove(state.pvMove),
			IncorrectMove: NewMove(nextMove),
		}
		log.Printf("Found puzzle: %+v\n", newpuzzle)
		state.puzzles = append(state.puzzles, newpuzzle)
	}
}

func processOutput(ioReader io.Reader, state *analysisState, done chan int) {
	reader := bufio.NewReader(ioReader)

	for {
		nextLine, err := reader.ReadBytes('\n')
		processOutputLine(string(nextLine), state)
		if err != nil {
			break
		}
	}

	done <- 0
}

func processOutputLine(line string, state *analysisState) {
	targetDepth := viper.GetInt("AnalysisDepth")

	// If the line matches 'readyok' mark the state as ready
	if readyRegex.Match([]byte(line)) {
		state.ready = true
		return
	}

	// If the line matches PV scores, record the scores and suggested move
	pvScores := pvScoreRegex.FindStringSubmatch(line)
	if pvScores != nil && len(pvScores) == 5 {
		score, _ := strconv.Atoi(pvScores[3])
		depth, _ := strconv.Atoi(pvScores[1])
		if pvScores[2] == "1" {
			state.pv1Score = score
			state.pv1Depth = depth
			state.pvMove = pvScores[4]
		} else if pvScores[2] == "2" {
			state.pv2Score = score
			state.pv2Depth = depth
		}
		if state.pv1Depth >= targetDepth && state.pv2Depth >= targetDepth {
			state.positionsAnalyzed <- state.positionID
		}
		return
	}

	// If the line matches PV mate, record mate scores and suggested move
	pvMateScores := pvMateRegex.FindStringSubmatch(line)
	if pvMateScores != nil && len(pvMateScores) == 5 {
		mateMoves, _ := strconv.Atoi(pvMateScores[3])
		score := 10000 * mateMoves / abs(mateMoves) // Treat a mating line as a 10,000 cp score
		depth, _ := strconv.Atoi(pvMateScores[1])
		if pvMateScores[2] == "1" {
			state.pv1Score = score
			state.pv1Depth = depth
			state.pvMove = pvMateScores[4]
		} else if pvMateScores[2] == "2" {
			state.pv2Score = score
			state.pv2Depth = depth
		}
		return
	}

	// If the line matches a FEN, record the FEN and remember the previous fen
	fen := fenRegex.FindStringSubmatch(line)
	if fen != nil && len(fen) == 2 {
		state.previousFen = state.fen
		state.fen = fen[1]
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func initializeEngineSession(input io.WriteCloser) {
	input.Write([]byte("uci\n"))
	input.Write([]byte("ucinewgame\n"))
	input.Write([]byte("isready\n"))
	input.Write([]byte("setoption name UCI_AnalyseMode value true\n"))
	input.Write([]byte("setoption name MultiPV value 2\n"))
}
