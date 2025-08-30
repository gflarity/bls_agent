package bls

import (
	"fmt"
	"time"

	"github.com/gflarity/bls_agent/pkg/bls"
	"go.temporal.io/sdk/workflow"
)

// WorkflowParams contains all the parameters needed for the BLSReleaseSummaryWorkflow
type WorkflowParams struct {
	Mins float64 `json:"mins"`
	// OpenAI configuration
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIBaseURL string `json:"openai_base_url"`
	OpenAIModel   string `json:"openai_model"`
	// Twitter credentials
	TwitterAPIKey       string `json:"twitter_api_key"`
	TwitterAPISecret    string `json:"twitter_api_secret"`
	TwitterAccessToken  string `json:"twitter_access_token"`
	TwitterAccessSecret string `json:"twitter_access_token_secret"`
}

// TweetResponse represents the expected response from the LLM
type TweetResponse struct {
	Tweet string `json:"tweet" jsonschema:"required,description=A single tweet summarizing the BLS release,minLength=1,maxLength=280"`
}

// BLSReleaseSummaryWorkflow is a workflow that generates BLS release summaries
func BLSReleaseSummaryWorkflow(ctx workflow.Context, params WorkflowParams) ([]string, error) {
	// Set workflow timeout
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 600 * time.Second,
	})

	// Execute FindEventsActivity to get BLS events
	var events []bls.Event
	err := workflow.ExecuteActivity(ctx, FindEventsActivity, params.Mins).Get(ctx, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to find events: %w", err)
	}

	// Log the number of events found
	workflow.GetLogger(ctx).Info("Found BLS events", "count", len(events))

	// Process the events and create a summary
	if len(events) == 0 {
		return nil, nil
	}

	// Process each event individually and post to Twitter
	// Note that we continue even if one event fails, because we want to post any tweets if possible
	var twtsums []string
	for i, event := range events {
		workflow.GetLogger(ctx).Info("Processing event", "index", i, "summary", event.Summary)

		var html string
		err := workflow.ExecuteActivity(ctx, FetchReleaseHTMLActivity, event).Get(ctx, &html)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to fetch HTML for event", "event", event.Summary, "error", err)
			// Continue with other events even if one fails
			continue
		}

		// Extract twtsum from HTML
		var txtsum string
		err = workflow.ExecuteActivity(ctx, ExtractSummaryActivity, html).Get(ctx, &txtsum)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to extract summary from HTML", "event", event.Summary, "error", err)
			continue
		}

		// Use LLM to create a Twitter-appropriate summary for this specific event
		prompt := fmt.Sprintf("Create a concise tweet summarizing this BLS release: %s\n\nContent: %s\n\nCreate a single engaging tweet under 280 characters focusing on the most important economic insights and data points.", event.Summary, txtsum)

		// Temporarily use a hardcoded schema string for testing
		schemaStr := `{"type":"object","properties":{"tweet":{"type":"string","description":"A single tweet summarizing the BLS release","minLength":1,"maxLength":280}},"required":["tweet"]}`

		// Log the hardcoded schema for debugging
		workflow.GetLogger(ctx).Info("Using hardcoded schema string",
			"schemaStr", schemaStr,
			"schemaStrType", fmt.Sprintf("%T", schemaStr))

		// Get LLM configuration from workflow params
		apiKey := params.OpenAIAPIKey
		baseURL := params.OpenAIBaseURL
		model := params.OpenAIModel

		sysprom := "You are an expert economic analyst who creates engaging single tweets about BLS (Bureau of Labor Statistics) releases. Your responses must follow the exact JSON schema provided."

		// Final validation of all parameters before activity call
		workflow.GetLogger(ctx).Debug("Final parameters for CompleteWithSchemaActivity",
			"baseURL", baseURL,
			"schemaStr", schemaStr,
			"systemPrompt", sysprom,
			// only print the first 80 characters of the prompt
			"prompt", prompt,
			"model", model,
			"apiKeyType", fmt.Sprintf("%T", apiKey),
			"baseURLType", fmt.Sprintf("%T", baseURL),
			"schemaStrType", fmt.Sprintf("%T", schemaStr),
			"systemPromptType", fmt.Sprintf("%T", sysprom),
			"promptType", fmt.Sprintf("%T", prompt),
			"modelType", fmt.Sprintf("%T", model))

		var twtsum string
		err = workflow.ExecuteActivity(ctx, CompleteWithSchemaActivity, apiKey, baseURL, schemaStr, sysprom, prompt, model).Get(ctx, &twtsum)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to create Twitter summary with LLM for event", "event", event.Summary, "error", err)
			continue

		}
		twtsums = append(twtsums, twtsum)

		/*
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

				err = workflow.ExecuteActivity(ctx, PostTweetActivity, tweetText, params.TwitterAPIKey, params.TwitterAPISecret, params.TwitterAccessToken, params.TwitterAccessSecret).Get(ctx, nil)
				if err != nil {
					workflow.GetLogger(ctx).Error("Failed to post tweet for event", "event", event.Summary, "tweet", tweetText, "error", err)
					// Continue with other events even if tweeting fails
				} else {
					workflow.GetLogger(ctx).Info("Successfully posted tweet for event", "event", event.Summary, "tweet", tweetText[:min(len(tweetText), 50)])
				}
			} else {
				workflow.GetLogger(ctx).Warn("No valid tweet generated for event", "event", event.Summary)
			}

		*/
	}

	// Combine all summaries
	finalSummary := fmt.Sprintf("BLS Release Summary (%d events processed):\n\n", len(twtsums))
	for i, summary := range twtsums {
		finalSummary += fmt.Sprintf("--- Event %d ---\n%s\n\n", i+1, summary)
	}

	return twtsums, nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
