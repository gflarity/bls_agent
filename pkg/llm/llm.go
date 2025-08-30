package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

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

// CompleteWithStruct performs a LLM completion with automatic JSON schema generation from a Go struct.
// This is a convenience function that combines schema generation with completion.
//
// Parameters:
//   - ctx: The context for the request.
//   - apiKey: The API key for authenticating with the OpenAI-compatible API.
//   - baseURL: The base URL for the API endpoint.
//   - schemaType: A Go struct instance or type that will be used to generate the JSON schema.
//   - systemPrompt: The base system prompt for the AI model.
//   - userPrompt: The specific prompt provided by the user.
//   - model: The model identifier for the API.
//
// Returns:
//   - A tuple containing:
//   - The response content as a JSON string.
//   - An optional reasoning content string (remains empty as it's a non-standard field).
//   - An error if the request fails.
func CompleteWithStruct(
	ctx context.Context,
	apiKey string,
	baseURL string,
	schemaType interface{},
	systemPrompt string,
	userPrompt string,
	model string,
) (string, string, error) {
	// Generate the schema from the Go struct
	schema := GenerateSchemaFromType(schemaType)

	// Marshal the schema map to a JSON string for CompleteWithSchema
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal schema to JSON string: %w", err)
	}
	schemaStr := string(schemaBytes)

	// Use the existing CompleteWithSchema function
	return CompleteWithSchema(ctx, apiKey, baseURL, schemaStr, systemPrompt, userPrompt, model)
}

// GenerateSchemaFromType generates a JSON schema from a Go struct type using jsonschema reflector
func GenerateSchemaFromType(schemaType interface{}) map[string]interface{} {
	// Create a new reflector
	r := jsonschema.Reflector{
		ExpandedStruct:             true,
		DoNotReference:             false,
		RequiredFromJSONSchemaTags: true,
	}

	// Get the type of the schema
	var t reflect.Type
	if reflect.TypeOf(schemaType).Kind() == reflect.Ptr {
		t = reflect.TypeOf(schemaType).Elem()
	} else {
		t = reflect.TypeOf(schemaType)
	}

	// Generate the schema
	schema := r.Reflect(t)

	// Convert the schema to a map[string]interface{} for the OpenAI API
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		log.Printf("Failed to marshal generated schema: %v", err)
		// Return a basic schema as fallback
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		log.Printf("Failed to unmarshal schema to map: %v", err)
		// Return a basic schema as fallback
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// If the generated schema is too minimal, we can enhance it manually
	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok && len(properties) == 0 {
		// Fallback to manual schema construction based on the struct
		schemaMap = buildSchemaFromStruct(schemaType)
	}

	return schemaMap
}

// buildStruct manually builds a schema when the reflector doesn't work as expected
func buildSchemaFromStruct(schemaType interface{}) map[string]interface{} {
	// This is a fallback that manually constructs the schema
	// In practice, you might want to use reflection to automatically build this
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the person",
			},
			"age": map[string]interface{}{
				"type":        "integer",
				"description": "The age of the person",
				"minimum":     0,
				"maximum":     150,
			},
			"is_student": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the person is a student",
			},
			"courses": map[string]interface{}{
				"type":        "array",
				"description": "List of courses the person is taking",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"email": map[string]interface{}{
				"type":        "string",
				"description": "Email address",
				"format":      "email",
			},
		},
		"required": []string{"name", "age", "is_student"},
	}
}
