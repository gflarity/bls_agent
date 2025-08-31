package bls

import (
	"context"
	"fmt"

	"github.com/gflarity/bls_agent/pkg/bls"
	"github.com/gflarity/bls_agent/pkg/llm"
	"github.com/gflarity/bls_agent/pkg/twitter"
	"go.temporal.io/sdk/activity"
)

// FindEventsActivity finds BLS events that happened within the last specified minutes
func FindEventsActivity(ctx context.Context, mins float64) ([]bls.Event, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing FindEventsActivity",
		"workflowID", workflowID,
		"runID", runID,
		"mins", mins)

	// Call the BLS package function
	events, err := bls.FindEvents(mins)
	if err != nil {
		activity.GetLogger(ctx).Error("FindEventsActivity failed", "error", err)
		return nil, fmt.Errorf("failed to find events: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("FindEventsActivity completed successfully",
		"eventsFound", len(events))

	return events, nil
}

// CompleteWithSchemaActivity performs LLM completion with a specified JSON schema
func CompleteWithSchemaActivity(
	ctx context.Context,
	apiKey string,
	baseURL string,
	schema string,
	systemPrompt string,
	userPrompt string,
	model string,
) (string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing CompleteWithSchemaActivity",
		"workflowID", workflowID,
		"runID", runID,
		"model", model,
		"baseURL", baseURL)

	// Call the LLM package function
	content, reasoning, err := llm.CompleteWithSchema(
		ctx,
		apiKey,
		baseURL,
		schema,
		systemPrompt,
		userPrompt,
		model,
	)
	if err != nil {
		activity.GetLogger(ctx).Error("CompleteWithSchemaActivity failed", "error", err)
		return "", fmt.Errorf("failed to complete with schema: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("CompleteWithSchemaActivity completed successfully",
		"contentLength", len(content),
		"reasoningLength", len(reasoning))

	return content, nil
}

// FetchReleaseHTMLActivity fetches the HTML for the release of an event
func FetchReleaseHTMLActivity(ctx context.Context, event bls.Event) (string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing FetchReleaseHTMLActivity",
		"workflowID", workflowID,
		"runID", runID,
		"eventSummary", event.Summary)

	// Call the BLS package function
	html, err := bls.FetchReleaseHTML(event)
	if err != nil {
		activity.GetLogger(ctx).Error("FetchReleaseHTMLActivity failed", "error", err)
		return "", fmt.Errorf("failed to fetch release HTML: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("FetchReleaseHTMLActivity completed successfully",
		"htmlLength", len(html))

	return html, nil
}

// ExtractSummaryActivity extracts the summary text from the release HTML
func ExtractSummaryActivity(ctx context.Context, html string) (string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing ExtractSummaryActivity",
		"workflowID", workflowID,
		"runID", runID,
		"htmlLength", len(html))

	// Call the BLS package function
	summary, err := bls.ExtractSummary(html)
	if err != nil {
		activity.GetLogger(ctx).Error("ExtractSummaryActivity failed", "error", err)
		return "", fmt.Errorf("failed to extract summary from HTML: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("ExtractSummaryActivity completed successfully",
		"summaryLength", len(summary))

	return summary, nil
}

// PostTweetThreadActivity posts a thread of tweets to Twitter
func PostTweetThreadActivity(ctx context.Context, tweetTexts []string, twitterAPIKey, twitterAPISecret, twitterAccessToken, twitterAccessSecret string) error {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing PostTweetThreadActivity",
		"workflowID", workflowID,
		"runID", runID,
		"tweetCount", len(tweetTexts))

	// Create a new Twitter client with provided credentials
	client, err := twitter.NewClientWithCredentials(twitterAPIKey, twitterAPISecret, twitterAccessToken, twitterAccessSecret)
	if err != nil {
		activity.GetLogger(ctx).Error("PostTweetThreadActivity failed to create Twitter client", "error", err)
		return fmt.Errorf("failed to create Twitter client: %w", err)
	}

	// Post the tweet thread
	err = client.PostTweetThread(tweetTexts)
	if err != nil {
		activity.GetLogger(ctx).Error("PostTweetThreadActivity failed to post thread", "error", err)
		return fmt.Errorf("failed to post tweet thread: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("PostTweetThreadActivity completed successfully",
		"tweetsPosted", len(tweetTexts))

	return nil
}

// PostTweetActivity posts a single tweet to Twitter
func PostTweetActivity(ctx context.Context, tweetText string, twitterAPIKey, twitterAPISecret, twitterAccessToken, twitterAccessSecret string, forReal bool) error {

	if !forReal {
		activity.GetLogger(ctx).Info("PostTweetActivity completed successfully (but not for real)",
			"tweetPosted", tweetText)
		return nil
	}

	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing PostTweetActivity",
		"workflowID", workflowID,
		"runID", runID,
		"tweetLength", len(tweetText))

	// Create a new Twitter client with provided credentials
	client, err := twitter.NewClientWithCredentials(twitterAPIKey, twitterAPISecret, twitterAccessToken, twitterAccessSecret)
	if err != nil {
		activity.GetLogger(ctx).Error("PostTweetActivity failed to create Twitter client", "error", err)
		return fmt.Errorf("failed to create Twitter client: %w", err)
	}

	// Post the single tweet
	_, err = client.PostTweet(tweetText, "")
	if err != nil {
		activity.GetLogger(ctx).Error("PostTweetActivity failed to post tweet", "error", err)
		return fmt.Errorf("failed to post tweet: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("PostTweetActivity completed successfully",
		"tweetPosted", tweetText)

	return nil
}
