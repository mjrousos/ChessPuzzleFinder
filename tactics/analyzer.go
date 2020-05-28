package tactics

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
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
	ready       bool
	pvMove      string
	pv1Score    int
	pv1Depth    int
	pv2Score    int
	pv2Depth    int
	fen         string
	previousFen string
	puzzles     []Puzzle
}

// FindPuzzles analyzes a chess game represented by a list of UCI moves
// and returns tactical puzzles from the game (or an empty slice
// if there are no interesting tactical puzzles).
func FindPuzzles(ctx context.Context, moves []string) []Puzzle {
	log.Printf("Analyzing a game with %d moves\n", len(moves))

	cmd := exec.Command(viper.GetString("StockfishPath"))
	stockfishOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to pipe Stockfish output: %v\n", err)
	}
	stockfishIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to pipe Stockfish input: %v\n", err)
	}
	cmd.Start()

	state := analysisState{puzzles: make([]Puzzle, 0)}
	done := make(chan int)
	go processOutput(stockfishOut, &state, done)
	initializeStockfishSession(stockfishIn)

	// The first several moves are unlikely to reveal an interesting tactic, so start iterator at 6
	for i := 6; i < len(moves); i++ {
		select {
		case <-ctx.Done():
			// If the app is interrupted, cancel analysis
			break
		default:
			analyzeMove(stockfishIn, moves[:i], moves[i], &state)
		}
	}

	// Close Stockfish
	stockfishIn.Write([]byte("quit"))
	stockfishIn.Close()
	cmd.Wait()
	<-done

	return state.puzzles
}

func analyzeMove(input io.WriteCloser, moves []string, nextMove string, state *analysisState) {
	// Set position
	setPositionCommand := fmt.Sprintf("position startpos moves %v\n", strings.Join(moves, " "))
	input.Write([]byte(setPositionCommand))

	// Analyze
	input.Write([]byte("go infinite\n"))

	// Wait for analysis
	// TODO - This could be optimized to stop after 10 seconds or depth 22 on both PV, whichever comes first (or ,really, just on depth 20; it should take that long in most cases)
	time.Sleep(time.Duration(viper.GetInt("AnalysisSecondsPerMove")) * time.Second)

	// Stop analysis, get FEN, and wait to make sure stdout has been written to
	input.Write([]byte("stop\n"))
	input.Write([]byte("d\n"))
	time.Sleep(100 * time.Millisecond)

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

	log.Printf("Evaluated: %s\n", state.fen)
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

func initializeStockfishSession(input io.WriteCloser) {
	input.Write([]byte("uci\n"))
	input.Write([]byte("ucinewgame\n"))
	input.Write([]byte("isready\n"))
	input.Write([]byte("setoption name UCI_AnalyseMode value true\n"))
	input.Write([]byte("setoption name MultiPV value 2\n"))
}
