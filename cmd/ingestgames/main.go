package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/azure-storage-queue-go/azqueue"
	"github.com/mjrousos/ChessPuzzleFinder/tactics"
	"github.com/spf13/viper"
)

var production bool

func main() {
	configure()
	showHeader()
	storageAccountName := viper.GetString("StorageAccountName")
	storageAccountKey := viper.GetString("StorageAccountKey")
	queueName := viper.GetString("GameIngestionQueue")
	workerCount := viper.GetInt("WorkerCount")

	msgURL := getQueueMessageURL(storageAccountName, storageAccountKey, queueName)

	log.Printf("Processing games from queue \"%s\" with %d workers\n", queueName, workerCount)

	// Cancel work on SIGINT/SIGTERM via signal.NotifyContext (Go 1.16+).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := sync.WaitGroup{}
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go processMessages(ctx, &wg, msgURL)
	}

	<-ctx.Done()
	log.Println("Canceling game processing...")
	wg.Wait()
	log.Println("- Done -")
}

func processMessages(ctx context.Context, wg *sync.WaitGroup, msgURL azqueue.MessagesURL) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			processMessage(ctx, msgURL)
		}
	}
}

func processMessage(ctx context.Context, msgURL azqueue.MessagesURL) {
	response, err := msgURL.Dequeue(ctx, 1, 30*time.Second)
	if err != nil {
		log.Println("Error dequeueing message: ", err)
		return
	}

	for i := int32(0); i < response.NumMessages(); i++ {
		msg := response.Message(i)
		log.Printf("Received message %s (%d bytes)\n", msg.ID, len(msg.Text))
		dec := json.NewDecoder(strings.NewReader(msg.Text))
		var g game
		err := dec.Decode(&g)
		if err != nil {
			log.Println("Error decoding json: ", err)
			continue
		}
		log.Printf("Processing game %s\n", g.GameURL)
		msgURL.NewMessageIDURL(msg.ID).Delete(ctx, msg.PopReceipt)

		puzzles := tactics.FindPuzzles(ctx, g.Ucimoves)
		log.Printf("Identified %d puzzles\n", len(puzzles))
		writePuzzlesToDatabase(ctx, g, puzzles)
	}
}

// Shows app header
func showHeader() {
	log.Println("-----------------------")
	log.Println("- Chess Puzzle Finder -")
	log.Println("-----------------------")
	log.Println()
}

// Sets up configuration based on json-based configuration and env vars
func configure() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AutomaticEnv()

	// Look for config next to the exe and in the current working dir
	appPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	cwd, _ := os.Getwd()
	viper.AddConfigPath(appPath)
	viper.AddConfigPath(cwd)

	err := viper.ReadInConfig()
	if err != nil {

		log.Fatalf("Could not read config file from %s or %s\n", appPath, cwd)
	}

	production = "Development" == viper.GetString("Environment")
}

func getQueueMessageURL(storageAccountName string, storageAccountKey string, queueName string) azqueue.MessagesURL {
	credential, err := azqueue.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		log.Fatal("Error creating credentials: ", err)
	}

	url, err := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net/%s", storageAccountName, queueName))
	if err != nil {
		log.Fatal("Error parsing url: ", err)
	}

	return azqueue.NewQueueURL(*url, azqueue.NewPipeline(credential, azqueue.PipelineOptions{})).NewMessagesURL()
}
