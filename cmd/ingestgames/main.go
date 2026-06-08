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

// visibilityTimeoutSeconds is how long a dequeued message stays invisible to
// other workers before becoming eligible for re-delivery. Puzzle analysis is
// CPU-bound (Stockfish runs for AnalysisSecondsPerMove on every move from
// move 6 onward), so a typical game can take several minutes to process.
// 10 minutes covers most games at the default 12s/move; if a worker crashes
// mid-game the message will be retried by another worker after this window.
const visibilityTimeoutSeconds = int32(600)

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

// deleteMessageTimeout bounds how long we wait when deleting an
// already-processed (or malformed) message. The delete runs on a detached
// background context so SIGINT/SIGTERM during shutdown doesn't cause the
// final delete to fail with `context canceled`, which would re-deliver a
// message we've already finished processing and produce duplicate DB rows.
const deleteMessageTimeout = 10 * time.Second

// deleteMessage removes msg from the queue using a detached, short-deadline
// context so the call survives worker-context cancellation during shutdown.
func deleteMessage(queueClient *azqueue.QueueClient, messageID, popReceipt string) error {
	ctx, cancel := context.WithTimeout(context.Background(), deleteMessageTimeout)
	defer cancel()
	_, err := queueClient.DeleteMessage(ctx, messageID, popReceipt, nil)
	return err
}

// processMessage dequeues at most one message and returns true if at least
// one message was dequeued — including malformed or undecodable ones, which
// are deleted to drain them from the queue. Returning true here suppresses
// the worker's empty-queue backoff so polling stays responsive whenever the
// queue has *any* activity.
func processMessage(ctx context.Context, queueClient *azqueue.QueueClient) bool {
	resp, err := queueClient.DequeueMessages(ctx, &azqueue.DequeueMessagesOptions{
		NumberOfMessages:  to.Ptr(int32(1)),
		VisibilityTimeout: to.Ptr(visibilityTimeoutSeconds),
	})
	if err != nil {
		log.Println("Error dequeueing message: ", err)
		return false
	}
	if len(resp.Messages) == 0 {
		return false
	}

	for _, msg := range resp.Messages {
		// If MessageID or PopReceipt are missing we can't delete; just log
		// and continue. The message will eventually be re-delivered and may
		// be re-skipped — this branch is essentially "never happens" against
		// a real Azure Queue service.
		if msg.MessageID == nil || msg.PopReceipt == nil {
			log.Println("Skipping message with missing MessageID or PopReceipt")
			continue
		}
		messageID := *msg.MessageID
		popReceipt := *msg.PopReceipt

		// Drain malformed messages (nil body or undecodable JSON) so they
		// don't cycle as poison messages — the body won't become valid on
		// retry, so retrying just wedges the queue.
		if msg.MessageText == nil {
			log.Printf("Deleting message %s with nil MessageText\n", messageID)
			if err := deleteMessage(queueClient, messageID, popReceipt); err != nil {
				log.Println("Error deleting malformed message: ", err)
			}
			continue
		}
		body := *msg.MessageText
		log.Printf("Received message %s (%d bytes)\n", messageID, len(body))

		dec := json.NewDecoder(strings.NewReader(body))
		var g game
		if err := dec.Decode(&g); err != nil {
			log.Printf("Deleting message %s after JSON decode error: %v\n", messageID, err)
			if err := deleteMessage(queueClient, messageID, popReceipt); err != nil {
				log.Println("Error deleting undecodable message: ", err)
			}
			continue
		}
		log.Printf("Processing game %s\n", g.GameURL)

		puzzles := tactics.FindPuzzles(ctx, g.Ucimoves)
		log.Printf("Identified %d puzzles\n", len(puzzles))
		writePuzzlesToDatabase(ctx, g, puzzles)

		// If the worker context was canceled during analysis or DB write,
		// the work for this message may be incomplete (writePuzzlesToDatabase
		// returns early on ctx.Done; FindPuzzles likewise checks ctx between
		// moves). Skip the final delete so the message becomes eligible for
		// re-delivery after visibilityTimeoutSeconds — accepting possible
		// duplicate puzzle rows on retry rather than silently losing the
		// remaining work.
		if ctx.Err() != nil {
			log.Printf("Skipping delete of message %s: worker context canceled mid-processing\n", messageID)
			continue
		}

		// Delete only after analysis + DB write succeed. Uses a detached
		// context (see deleteMessage) so that a SIGINT/SIGTERM that arrives
		// *between* the ctx.Err check above and this call doesn't cause
		// the final delete to fail with `context canceled` and re-deliver
		// an already-completed message (which would otherwise produce
		// duplicate puzzle rows on retry).
		if err := deleteMessage(queueClient, messageID, popReceipt); err != nil {
			log.Println("Error deleting message: ", err)
		}
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
