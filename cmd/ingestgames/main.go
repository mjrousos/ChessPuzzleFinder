package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/v2"
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

	queueClient := newQueueClient(storageAccountName, storageAccountKey, queueName)

	log.Printf("Processing games from queue \"%s\" with %d workers\n", queueName, workerCount)

	// Cancel work on SIGINT/SIGTERM via signal.NotifyContext (Go 1.16+).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	wg := sync.WaitGroup{}
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go processMessages(ctx, &wg, queueClient)
	}

	<-ctx.Done()
	log.Println("Canceling game processing...")
	wg.Wait()
	log.Println("- Done -")
}

// emptyQueueBackoff is how long a worker waits before re-polling after a
// dequeue that returns no messages. Prevents hot-polling against the queue
// service when the queue is idle. Cancelable via the worker's context.
const emptyQueueBackoff = 5 * time.Second

func processMessages(ctx context.Context, wg *sync.WaitGroup, queueClient *azqueue.QueueClient) {
	defer wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		gotWork := processMessage(ctx, queueClient)
		if gotWork {
			continue
		}
		// Empty dequeue: back off, but stay cancelable so shutdown is prompt.
		select {
		case <-ctx.Done():
			return
		case <-time.After(emptyQueueBackoff):
		}
	}
}

// processMessage dequeues at most one message and returns true if a message
// was processed (so the caller knows whether to back off before re-polling).
func processMessage(ctx context.Context, queueClient *azqueue.QueueClient) bool {
	resp, err := queueClient.DequeueMessages(ctx, &azqueue.DequeueMessagesOptions{
		NumberOfMessages:  to.Ptr(int32(1)),
		VisibilityTimeout: to.Ptr(int32(30)),
	})
	if err != nil {
		log.Println("Error dequeueing message: ", err)
		return false
	}
	if len(resp.Messages) == 0 {
		return false
	}

	for _, msg := range resp.Messages {
		if msg.MessageText == nil || msg.MessageID == nil || msg.PopReceipt == nil {
			log.Println("Skipping malformed message with missing fields")
			continue
		}
		messageID := *msg.MessageID
		body := *msg.MessageText
		log.Printf("Received message %s (%d bytes)\n", messageID, len(body))

		dec := json.NewDecoder(strings.NewReader(body))
		var g game
		if err := dec.Decode(&g); err != nil {
			log.Println("Error decoding json: ", err)
			continue
		}
		log.Printf("Processing game %s\n", g.GameURL)
		if _, err := queueClient.DeleteMessage(ctx, messageID, *msg.PopReceipt, nil); err != nil {
			log.Println("Error deleting message: ", err)
		}

		puzzles := tactics.FindPuzzles(ctx, g.Ucimoves)
		log.Printf("Identified %d puzzles\n", len(puzzles))
		writePuzzlesToDatabase(ctx, g, puzzles)
	}
	return true
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

func newQueueClient(storageAccountName, storageAccountKey, queueName string) *azqueue.QueueClient {
	cred, err := azqueue.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		log.Fatal("Error creating credentials: ", err)
	}

	serviceURL := fmt.Sprintf("https://%s.queue.core.windows.net/", storageAccountName)
	svc, err := azqueue.NewServiceClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		log.Fatal("Error creating queue service client: ", err)
	}

	return svc.NewQueueClient(queueName)
}
