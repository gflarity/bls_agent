package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gflarity/bls_agent/internal/workflows/arxiv"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
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
		temporalNamespace = "default" // default
	}

	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "bls-agent" // default
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  temporalHostPort,
		Namespace: temporalNamespace,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Set the target date for paper search (default to today)
	targetDate := time.Now().AddDate(0, 0, -4)
	if dateStr := os.Getenv("TARGET_DATE"); dateStr != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = parsedDate
		} else {
			log.Printf("Warning: Invalid TARGET_DATE format '%s', using today's date", dateStr)
		}
	}

	// Create workflow parameters
	workflowParams := arxiv.PaperOfTheDayWorkflowParams{
		Date: targetDate,
		// OpenAI configuration
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
		OpenAIModel:   os.Getenv("OPENAI_MODEL"),
	}

	// Create workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        "paper-of-the-day-" + targetDate.Format("20060102") + "-" + time.Now().Format("150405"),
		TaskQueue: taskQueue,
	}

	// Start the Paper of the Day workflow
	log.Printf("Starting PaperOfTheDayWorkflow for date: %s...", targetDate.Format("2006-01-02"))
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, arxiv.PaperOfTheDayWorkflow, workflowParams)
	if err != nil {
		log.Fatalln("Unable to execute PaperOfTheDayWorkflow", err)
	}

	log.Printf("Started PaperOfTheDayWorkflow: %s, RunID: %s\n", we.GetID(), we.GetRunID())

	var res []string
	err = we.Get(context.Background(), &res)
	if err != nil {
		log.Fatalln("PaperOfTheDayWorkflow execution failed", err)
	}

	log.Printf("PaperOfTheDayWorkflow completed successfully. Result: %v\n", res)
	log.Println("Paper of the Day workflow completed!")
}
