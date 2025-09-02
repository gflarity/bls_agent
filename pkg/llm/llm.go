package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

// LLMResponseError is a custom error type for when the LLM fails to generate a response.
var LLMResponseError = errors.New("failed to generate a valid LLM response")

// CompleteWithSchema performs a LLM completion with a specified JSON schema using the official OpenAI Go library.
//
// Parameters:
//   - ctx: The context for the request.
//   - apiKey: The API key for authenticating with the OpenAI-compatible API.
//   - baseURL: The base URL for the API endpoint.
//   - schema: A JSON schema string that the response must follow.
//   - systemPrompt: The base system prompt for the AI model.
//   - userPrompt: The specific prompt provided by the user.
//   - model: The model identifier for the API.
//
// Returns:
//   - A tuple containing:
//   - The response content as a JSON string.
//   - An optional reasoning content string (remains empty as it's a non-standard field).
//   - An error if the request fails.
func CompleteWithSchema(
	ctx context.Context,
	apiKey string,
	baseURL string,
	schema string,
	systemPrompt string,
	userPrompt string,
	model string,
) (string, string, error) {
	// Initialize the OpenAI client using the official library's pattern.
	// We use `option.WithBaseURL` to specify a custom endpoint.
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	// Schema is already a JSON string, no need to marshal
	schemaStr := schema

	// Unmarshal the JSON schema string back to a map for the OpenAI API
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &schemaMap); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal schema string to map: %w", err)
	}

	// Construct the system message.
	systemMessage := fmt.Sprintf("%s Here's the json schema you need to adhere to: <schema>%s</schema>", systemPrompt, schemaStr)

	// Create the chat completion request using the official library's builder-style API.
	// The parameters are passed in a `ChatCompletionNewParams` struct.
	params := openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMessage),
			openai.UserMessage(userPrompt),
		},
		// The response format structure is slightly different in the official library.
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				Type: "json_schema",
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "response",
					Schema: schemaMap,
					Strict: openai.Bool(true),
				},
			},
		},
	}

	// The official library's client methods are organized by API resource (e.g., Chat, Images).
	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("chat completion request failed: %w", err)
	}

	// Check if the response contains any choices and content.
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		fullResponse, _ := json.MarshalIndent(resp, "", "  ")
		log.Printf("Received an empty or invalid response from the API: %s\n", string(fullResponse))
		return "", "", LLMResponseError
	}

	content := resp.Choices[0].Message.Content
	reasoning := "" // This non-standard field is not available in the official library.

	return content, reasoning, nil
}

// GenerateSchemaFromType generates a JSON schema from a Go struct type using jsonschema reflector
func GenerateSchemaFromType(schemaType interface{}) (map[string]interface{}, error) {
	// Create a new reflector with settings optimized for OpenAI API compatibility
	r := jsonschema.Reflector{
		// DoNotReference: true ensures we get an inline schema instead of $ref
		DoNotReference: true,
		// RequiredFromJSONSchemaTags: true respects jsonschema tags for required fields
		RequiredFromJSONSchemaTags: true,
	}

	// Pass the actual value/pointer to Reflect(), not the reflect.Type
	// The jsonschema.Reflector.Reflect() method expects an actual value or pointer
	schema := r.Reflect(schemaType)

	// Convert the schema to a map[string]interface{} for the OpenAI API
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generated schema: %w", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema to map: %w", err)
	}

	// By default, make all fields required unless explicitly marked as optional
	// This is typical for OpenAI API usage where we want complete responses
	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok && len(properties) > 0 {
		if _, hasRequired := schemaMap["required"]; !hasRequired {
			var required []string
			for fieldName := range properties {
				required = append(required, fieldName)
			}
			schemaMap["required"] = required
		}
	}

	return schemaMap, nil
}

func GenerateSchema(schemaType interface{}) (string, error) {
	schema, err := GenerateSchemaFromType(schemaType)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema from type: %w", err)
	}
	// Pretty print the generated schema
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Marshal the schema map to a JSON string for CompleteWithSchema
	return string(schemaBytes), nil
}
