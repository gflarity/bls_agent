package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gflarity/bls_agent/internal/config"
	"github.com/gflarity/bls_agent/internal/workflows/bls"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  cfg.TemporalHostPort,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, cfg.TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(bls.BLSReleaseSummaryWorkflow)

	// Register activities
	w.RegisterActivity(bls.FindEventsActivity)
	w.RegisterActivity(bls.FetchReleaseHTMLActivity)
	w.RegisterActivity(bls.ExtractSummaryActivity)

	// Start worker
	log.Printf("Starting worker on task queue: %s", cfg.TaskQueue)
	log.Printf("Connected to Temporal server: %s", cfg.TemporalHostPort)
	log.Printf("Namespace: %s", cfg.TemporalNamespace)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping worker...")
		w.Stop()
	}()

	// Run worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("Unable to start worker", err)
	}

	log.Println("Worker stopped")
}
