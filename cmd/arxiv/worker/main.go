package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gflarity/bls_agent/internal/workflows/arxiv"
	"github.com/joho/godotenv"

	"go.temporal.io/sdk/client"
	temporallog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
)

func main() {

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		// Continue execution as environment variables might be set elsewhere
	}

	// Get Temporal configuration from environment
	temporalHostPort := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHostPort == "" {
		temporalHostPort = "localhost:7233" // default
	}

	temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")
	if temporalNamespace == "" {
		panic("TEMPORAL_NAMESPACE environment variable is required")
	}

	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		panic("TEMPORAL_TASK_QUEUE environment variable is required")
	}

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
		HostPort:  temporalHostPort,
		Namespace: temporalNamespace,
		Logger:    temporalLogger,
	})
	if err != nil {
		panic(fmt.Errorf("unable to create Temporal client: %w", err))
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(arxiv.PaperOfTheDayWorkflow)

	// Register activities
	w.RegisterActivity(arxiv.GetArxivIdsForDateActivity)
	w.RegisterActivity(arxiv.GetArxivAbstractActivity)
	w.RegisterActivity(arxiv.ExtractPaperTextActivity)
	w.RegisterActivity(arxiv.CompleteWithSchemaActivity)

	// Start worker
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("Received shutdown signal, stopping worker...\n")
		w.Stop()
	}()

	log.Printf("Starting ArXiv worker on task queue: %s", taskQueue)
	log.Printf("Temporal server: %s, namespace: %s", temporalHostPort, temporalNamespace)

	// Run worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		panic(fmt.Errorf("unable to start worker: %w", err))
	}

	log.Println("Worker stopped")
}
