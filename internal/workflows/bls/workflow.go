package bls

import (
	"fmt"
	"time"

	"github.com/apognu/gocal"
	"go.temporal.io/sdk/workflow"
)

// TweetResponse represents the expected response from the LLM
type TweetResponse struct {
	Tweet string `json:"tweet" jsonschema:"required,description=A single tweet summarizing the BLS release,minLength=1,maxLength=280"`
}

// BLSReleaseSummaryWorkflow is a workflow that generates BLS release summaries
func BLSReleaseSummaryWorkflow(ctx workflow.Context, mins float64) (string, error) {
	// Set workflow timeout
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 600 * time.Second,
	})

	// Execute FindEventsActivity to get BLS events
	var events []gocal.Event
	err := workflow.ExecuteActivity(ctx, FindEventsActivity, mins).Get(ctx, &events)
	if err != nil {
		return "", fmt.Errorf("failed to find events: %w", err)
	}

	// Log the number of events found
	workflow.GetLogger(ctx).Info("Found BLS events", "count", len(events))

	// Process the events and create a summary
	if len(events) == 0 {
		return "No BLS events found in the specified time window", nil
	}

	// Process each event individually and post to Twitter
	var txtSums []string
	for i, event := range events {
		workflow.GetLogger(ctx).Info("Processing event", "index", i, "summary", event.Summary)

		var html string
		err := workflow.ExecuteActivity(ctx, FetchReleaseHTMLActivity, event).Get(ctx, &html)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to fetch HTML for event", "event", event.Summary, "error", err)
			// Continue with other events even if one fails
			continue
		}

		// Extract summary from HTML
		var summary string
		err = workflow.ExecuteActivity(ctx, ExtractSummaryActivity, html).Get(ctx, &summary)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to extract summary from HTML", "event", event.Summary, "error", err)
			// Use a fallback summary if extraction fails
			summary = fmt.Sprintf("Event: %s (HTML extraction failed)", event.Summary)
		}

		txtSums = append(txtSums, summary)
		/*
			// Use LLM to create a Twitter-appropriate summary for this specific event
			var twitterSummary string
			prompt := fmt.Sprintf("Create a concise tweet summarizing this BLS release: %s\n\nContent: %s\n\nCreate a single engaging tweet under 280 characters focusing on the most important economic insights and data points.", event.Summary, summary)

			// Generate JSON schema dynamically from struct
			reflector := jsonschema.Reflector{
				ExpandedStruct:             true,
				DoNotReference:             true,
				RequiredFromJSONSchemaTags: true,
			}
			schema := reflector.Reflect(&TweetResponse{})

			// Get LLM configuration from environment or use defaults
			apiKey := os.Getenv("OPENAI_API_KEY")
			baseURL := os.Getenv("OPENAI_BASE_URL")
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			model := os.Getenv("OPENAI_MODEL")
			if model == "" {
				model = "gpt-4"
			}

			systemPrompt := "You are an expert economic analyst who creates engaging single tweets about BLS (Bureau of Labor Statistics) releases. Your responses must follow the exact JSON schema provided."

			err = workflow.ExecuteActivity(ctx, CompleteWithSchemaActivity, apiKey, baseURL, schema, systemPrompt, prompt, model).Get(ctx, &twitterSummary)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to create Twitter summary with LLM for event", "event", event.Summary, "error", err)
				// Fallback to basic summary
				twitterSummary = summary
			}

			// Process the LLM response for this event
			var tweetText string
			if twitterSummary != "" {
				// Parse the JSON response from LLM using the struct
				var response TweetResponse

				err = json.Unmarshal([]byte(twitterSummary), &response)
				if err != nil {
					return "", fmt.Errorf("failed to parse LLM response as JSON: %w", err)
				}

				// Validate tweet length
				if len(response.Tweet) > 280 {
					return "", fmt.Errorf("LLM generated tweet is too long: %d characters (max 280)", len(response.Tweet))
				}

				tweetText = response.Tweet
			}

			// Post the tweet for this specific event
			if tweetText != "" {
				workflow.GetLogger(ctx).Info("Posting tweet for event", "event", event.Summary, "tweetLength", len(tweetText))

				err = workflow.ExecuteActivity(ctx, PostTweetActivity, tweetText).Get(ctx, nil)
				if err != nil {
					workflow.GetLogger(ctx).Error("Failed to post tweet for event", "event", event.Summary, "tweet", tweetText, "error", err)
					// Continue with other events even if tweeting fails
				} else {
					workflow.GetLogger(ctx).Info("Successfully posted tweet for event", "event", event.Summary, "tweet", tweetText[:min(len(tweetText), 50)])
				}
			} else {
				workflow.GetLogger(ctx).Warn("No valid tweet generated for event", "event", event.Summary)
			}

			txtSums = append(txtSums, summary)
		*/
	}

	// Create final summary for return
	if len(txtSums) == 0 {
		return "Failed to process any BLS events", nil
	}

	// Combine all summaries
	finalSummary := fmt.Sprintf("BLS Release Summary (%d events processed):\n\n", len(txtSums))
	for i, summary := range txtSums {
		finalSummary += fmt.Sprintf("--- Event %d ---\n%s\n\n", i+1, summary)
	}

	return finalSummary, nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
