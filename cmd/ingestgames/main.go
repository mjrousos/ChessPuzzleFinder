package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mjrousos/ChessPuzzleFinder/tactics"
	"github.com/spf13/viper"
)

func main() {
	showHeader()
	configure()

	queueName := viper.GetString("GameIngestionQueue")
	fmt.Printf("Processing games from queue %v\n", queueName)
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
	viper.SetConfigType("json")
	viper.AutomaticEnv()

	// Look for config next to the exe and in the current working dir
	appPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	viper.AddConfigPath(appPath)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Could not read config file")
		panic(err)
	}
}
