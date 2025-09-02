package arxiv

import (
	"context"
	"fmt"
	"time"

	"github.com/gflarity/bls_agent/pkg/arxiv"
	"github.com/gflarity/bls_agent/pkg/llm"
	"go.temporal.io/sdk/activity"
)

// GetArxivIdsForDateActivity scrapes the Arxiv "recent" page to find all paper IDs
// published on a specific target date for the cs.AI category.
func GetArxivIdsForDateActivity(ctx context.Context, targetDate time.Time) ([]string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing GetArxivIdsForDateActivity",
		"workflowID", workflowID,
		"runID", runID,
		"targetDate", targetDate.Format("2006-01-02"))

	// Call the arxiv package function
	arxivIds, err := arxiv.GetArxivIdsForDate(targetDate)
	if err != nil {
		activity.GetLogger(ctx).Error("GetArxivIdsForDateActivity failed", "error", err)
		return nil, fmt.Errorf("failed to get arxiv IDs for date: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("GetArxivIdsForDateActivity completed successfully",
		"arxivIdsFound", len(arxivIds),
		"targetDate", targetDate.Format("2006-01-02"))

	return arxivIds, nil
}

// GetArxivAbstractActivity fetches and cleans the abstract text from a paper's abstract page.
func GetArxivAbstractActivity(ctx context.Context, arxivId string) (string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing GetArxivAbstractActivity",
		"workflowID", workflowID,
		"runID", runID,
		"arxivId", arxivId)

	// Call the arxiv package function
	abstract, err := arxiv.GetArxivAbstract(arxivId)
	if err != nil {
		activity.GetLogger(ctx).Error("GetArxivAbstractActivity failed", "error", err)
		return "", fmt.Errorf("failed to get arxiv abstract: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("GetArxivAbstractActivity completed successfully",
		"arxivId", arxivId,
		"abstractLength", len(abstract))

	return abstract, nil
}

// ExtractPaperTextActivity fetches a paper's PDF and extracts its plain text content.
func ExtractPaperTextActivity(ctx context.Context, arxivId string) (string, error) {
	// Get activity info
	activityInfo := activity.GetInfo(ctx)
	workflowID := activityInfo.WorkflowExecution.ID
	runID := activityInfo.WorkflowExecution.RunID

	// Log activity execution
	activity.GetLogger(ctx).Info("Executing ExtractPaperTextActivity",
		"workflowID", workflowID,
		"runID", runID,
		"arxivId", arxivId)

	// Call the arxiv package function
	text, err := arxiv.ExtractPaperText(arxivId)
	if err != nil {
		activity.GetLogger(ctx).Error("ExtractPaperTextActivity failed", "error", err)
		return "", fmt.Errorf("failed to extract paper text: %w", err)
	}

	// Log the results
	activity.GetLogger(ctx).Info("ExtractPaperTextActivity completed successfully",
		"arxivId", arxivId,
		"textLength", len(text))

	return text, nil
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
