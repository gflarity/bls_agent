package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gflarity/bls_agent/internal/workflows/bls"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		// Continue execution as environment variables might be set elsewhere
	}

	// Create workflow parameters with credentials from environment
	workflowParams := bls.WorkflowParams{
		Mins: 3 * 1440.0, // 1440. = 24 hours in minutes
		// OpenAI configuration
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
		OpenAIModel:   os.Getenv("OPENAI_MODEL"),
		// Twitter credentials
		TwitterAPIKey:       os.Getenv("X_API_KEY"),
		TwitterAPISecret:    os.Getenv("X_API_SECRET"),
		TwitterAccessToken:  os.Getenv("X_ACCESS_TOKEN"),
		TwitterAccessSecret: os.Getenv("X_ACCESS_TOKEN_SECRET"),

		// Tweet For Real
		TweetForReal: os.Getenv("TWEET_FOR_REAL") == "true",
	}

	// Validate required environment variables
	if workflowParams.OpenAIAPIKey == "" {
		log.Fatalln("OPENAI_API_KEY environment variable is required")
	}
	if workflowParams.TwitterAPIKey == "" || workflowParams.TwitterAPISecret == "" ||
		workflowParams.TwitterAccessToken == "" || workflowParams.TwitterAccessSecret == "" {
		log.Fatalln("All Twitter credentials (X_API_KEY, X_API_SECRET, X_ACCESS_TOKEN, X_ACCESS_TOKEN_SECRET) are required")
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  os.Getenv("TEMPORAL_HOST_PORT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// Create workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        "bls-release-summary-" + time.Now().Format("20060102-150405"),
		TaskQueue: os.Getenv("TEMPORAL_TASK_QUEUE"),
	}

	// Start the BLS Release Summary workflow
	log.Println("Starting BLSReleaseSummaryWorkflow...")
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, bls.BLSReleaseSummaryWorkflow, workflowParams)
	if err != nil {
		log.Fatalln("Unable to execute BLSReleaseSummaryWorkflow", err)
	}

	log.Printf("Started BLSReleaseSummaryWorkflow: %s, RunID: %s\n", we.GetID(), we.GetRunID())

	// Wait for workflow completion
	var twtsums []string
	err = we.Get(context.Background(), &twtsums)
	if err != nil {
		log.Fatalln("BLSReleaseSummaryWorkflow execution failed", err)
	}

	log.Printf("BLSReleaseSummaryWorkflow completed successfully. Result: %v\n", twtsums)
	log.Println("BLS Release Summary workflow completed!")
}
