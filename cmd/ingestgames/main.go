package main

import (
	"fmt"

	"github.com/mjrousos/ChessPuzzleFinder/tactics"
)

func main() {
	showHeader()
	configure()

	puzzles := tactics.FindPuzzles(nil)
	fmt.Printf("Identified %v puzzles\n", len(puzzles))

	fmt.Println("- Done -")
}

func showHeader() {
	fmt.Println("-----------------------")
	fmt.Println("- Chess Puzzle Finder -")
	fmt.Println("-----------------------")
	fmt.Println()
}

func configure() {
	viper.SetConfigName("config")
}
