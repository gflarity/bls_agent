package bls

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gflarity/bls_agent/pkg/bls"
	"github.com/gflarity/bls_agent/pkg/llm"
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

	// Tweet For Real
	TweetForReal bool `json:"tweet_for_real"`
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

		// create a ticker for 5 seconds, so we don't post tweets to quickly, we'll
		// wait below until the ticker is done
		timer := workflow.NewTimer(ctx, 5*time.Second)

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
		twtstruct := TweetResponse{}
		schema, err := llm.GenerateSchemaFromType(twtstruct)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to generate schema from type", "error", err)
			continue
		}

		// Pretty print the generated schema
		schemaBytes, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to marshal schema", "error", err)
			continue
		}

		// Marshal the schema map to a JSON string for CompleteWithSchema
		schemaStr := string(schemaBytes)
		workflow.GetLogger(ctx).Info("Generated Schema: %s", schemaStr)

		//schemaStr := `{"type":"object","properties":{"tweet":{"type":"string","description":"A single tweet summarizing the BLS release","minLength":1,"maxLength":280}},"required":["tweet"]}`

		// Get LLM configuration from workflow params
		apiKey := params.OpenAIAPIKey
		baseURL := params.OpenAIBaseURL
		model := params.OpenAIModel

		sysprom := "You are an expert economic analyst who creates engaging single tweets about BLS (Bureau of Labor Statistics) releases. Your responses must follow the exact JSON schema provided."

		// Final validation of all parameters before activity call
		var resp string
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

		err = workflow.ExecuteActivity(ctx, CompleteWithSchemaActivity, apiKey, baseURL, schemaStr, sysprom, prompt, model).Get(ctx, &resp)
		// unmarshal the response into the twtstruct
		err = json.Unmarshal([]byte(resp), &twtstruct)
		if err != nil {
			workflow.GetLogger(ctx).Error("Failed to unmarshal response into twtstruct", "error", err)
			workflow.GetLogger(ctx).Error("response", "response", resp)
			continue
		}

		// Process the LLM response for this event
		var twttxt string
		if resp != "" {
			err = json.Unmarshal([]byte(resp), &twtstruct)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to unmarshal response into twtstruct", "error", err)
				continue
			}
			twttxt = twtstruct.Tweet

			// Validate tweet length
			if len(twttxt) > 280 {
				workflow.GetLogger(ctx).Error("LLM generated tweet is too long: %d characters (max 280)", twttxt)
				continue
			}

		}

		// wait for the timer to finish so that we don't post tweets to quickly			// wait for the ticker to finish so that we don't post tweets to quickly
		timer.Get(ctx, nil)

		// Post the tweet for this specific event
		if twttxt != "" {
			workflow.GetLogger(ctx).Info("Posting tweet for event", "event", event.Summary, "tweetLength", len(twttxt))

			err = workflow.ExecuteActivity(ctx, PostTweetActivity, twttxt, params.TwitterAPIKey, params.TwitterAPISecret, params.TwitterAccessToken, params.TwitterAccessSecret, false).Get(ctx, nil)
			if err != nil {
				workflow.GetLogger(ctx).Error("Failed to post tweet for event", "event", event.Summary, "tweet", twttxt, "error", err)
				continue
			} else {
				workflow.GetLogger(ctx).Info("Successfully posted tweet for event", "event", event.Summary, "tweet", twttxt[:min(len(twttxt), 50)])
				twtsums = append(twtsums, twttxt)
			}
		} else {
			workflow.GetLogger(ctx).Error("No valid tweet generated for event", "event", event.Summary)
			continue
		}
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
