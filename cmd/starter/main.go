package main

import (
	"context"
	"log"
	"time"

	"github.com/gflarity/bls_agent/internal/config"
	"github.com/gflarity/bls_agent/internal/workflows/bls"
	"go.temporal.io/sdk/client"
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

	// Create workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        "bls-release-summary-" + time.Now().Format("20060102-150405"),
		TaskQueue: cfg.TaskQueue,
	}

	// Start the BLS Release Summary workflow
	log.Println("Starting BLSReleaseSummaryWorkflow...")
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, bls.BLSReleaseSummaryWorkflow, 1440.0)
	if err != nil {
		log.Fatalln("Unable to execute BLSReleaseSummaryWorkflow", err)
	}

	log.Printf("Started BLSReleaseSummaryWorkflow: %s, RunID: %s\n", we.GetID(), we.GetRunID())

	// Wait for workflow completion
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("BLSReleaseSummaryWorkflow execution failed", err)
	}

	log.Printf("BLSReleaseSummaryWorkflow completed successfully. Result: %s\n", result)
	log.Println("BLS Release Summary workflow completed!")
}
