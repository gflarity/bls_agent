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
		Mins: 5, //
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

	// Create schedule ID with timestamp to make it unique
	scheduleID := "bls-release-summary-cron-" + time.Now().Format("20060102-150405")

	// Create the schedule for daily execution at 8:30am and 10:00am Eastern Time
	log.Println("Creating BLS Release Summary cron schedule...")
	_, err = c.ScheduleClient().Create(context.Background(), client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			// Cron expressions for 8:30am and 10:00am
			CronExpressions: []string{
				"32 8 * * *", // 8:30 AM
				"2 10 * * *", // 10:00 AM
			},
			// Use Eastern Time zone to handle DST automatically
			TimeZoneName: "America/New_York",
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        "bls-release-summary-scheduled",
			TaskQueue: os.Getenv("TEMPORAL_TASK_QUEUE"),
			Workflow:  bls.BLSReleaseSummaryWorkflow,
			Args:      []interface{}{workflowParams},
		},
	})
	if err != nil {
		log.Fatalln("Unable to create schedule", err)
	}

	log.Printf("Successfully created BLS Release Summary cron schedule: %s\n", scheduleID)
	log.Println("The workflow will run daily at 8:30 AM and 10:00 AM Eastern Time")
	log.Println("Schedule created successfully!")
}
