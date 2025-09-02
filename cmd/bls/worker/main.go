package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gflarity/bls_agent/internal/workflows/bls"

	"go.temporal.io/sdk/client"
	temporallog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
)

func main() {

	// Configure logger to suppress debug logs
	// Create an slog logger with INFO level and above (suppresses DEBUG)
	// This will show: INFO, WARN, ERROR logs but suppress DEBUG logs
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // This suppresses DEBUG level logs
	}))

	// Create Temporal logger from slog
	temporalLogger := temporallog.NewStructuredLogger(slogLogger)

	// Create Temporal client with custom logger
	c, err := client.Dial(client.Options{
		HostPort:  os.Getenv("TEMPORAL_HOST_PORT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
		Logger:    temporalLogger,
	})
	if err != nil {
		panic(fmt.Errorf("Unable to create Temporal client: %w", err))
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, os.Getenv("TEMPORAL_TASK_QUEUE"), worker.Options{})

	// Register workflows
	w.RegisterWorkflow(bls.BLSReleaseSummaryWorkflow)

	// Register activities
	w.RegisterActivity(bls.FindEventsActivity)
	w.RegisterActivity(bls.FetchReleaseHTMLActivity)
	w.RegisterActivity(bls.ExtractSummaryActivity)
	w.RegisterActivity(bls.CompleteWithSchemaActivity)
	w.RegisterActivity(bls.PostTweetActivity)

	// Start worker
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("Received shutdown signal, stopping worker...\n")
		w.Stop()
	}()

	// Run worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		panic(fmt.Errorf("Unable to start worker: %w", err))
	}

	log.Println("Worker stopped")
}
