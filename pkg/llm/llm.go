package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

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
//   - schema: A map representing the JSON schema that the response must follow.
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
	schema map[string]interface{},
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

	// Marshal the schema map into a JSON string.
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}
	schemaStr := string(schemaBytes)

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
					Schema: schema,
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
	schema := generateSchemaFromType(schemaType)

	// Use the existing CompleteWithSchema function
	return CompleteWithSchema(ctx, apiKey, baseURL, schema, systemPrompt, userPrompt, model)
}

// generateSchemaFromType uses jsonschema reflector to generate a JSON schema from a Go struct type
func generateSchemaFromType(schemaType interface{}) map[string]interface{} {
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

	// If we got an empty struct instance, try to get the type from the reflect.Type
	if t.Kind() == reflect.Struct {
		// Check if this is an empty struct (no fields)
		if t.NumField() == 0 {
			log.Printf("Warning: Empty struct detected, this might cause issues with schema generation")
		}
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

	// Debug: log the generated schema
	log.Printf("Generated schema from type %v: %+v", t, schemaMap)

	// Check if the schema is valid (has properties)
	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
		if len(properties) == 0 {
			log.Printf("Warning: Generated schema has no properties, this might cause issues")
			// Try to build a schema manually as fallback
			return buildSchemaFromStructType(t)
		}
	}

	return schemaMap
}

// buildSchemaFromStructType manually builds a schema when the reflector doesn't work as expected
func buildSchemaFromStructType(t reflect.Type) map[string]interface{} {
	log.Printf("Building manual schema for type: %v", t)

	properties := make(map[string]interface{})
	required := []string{}

	// Iterate through struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag to get the field name
		fieldName := jsonTag
		if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
			fieldName = jsonTag[:commaIdx]
		}

		// Get jsonschema tag for additional metadata
		jsonschemaTag := field.Tag.Get("jsonschema")

		// Build field schema based on type
		fieldSchema := buildFieldSchema(field.Type, jsonschemaTag)
		properties[fieldName] = fieldSchema

		// Check if field is required (you could add logic here based on tags)
		required = append(required, fieldName)
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// buildFieldSchema builds a schema for a specific field type
func buildFieldSchema(fieldType reflect.Type, jsonschemaTag string) map[string]interface{} {
	schema := make(map[string]interface{})

	// Handle different types
	switch fieldType.Kind() {
	case reflect.String:
		schema["type"] = "string"
		// Parse jsonschema tag for format, description, etc.
		if strings.Contains(jsonschemaTag, "format=email") {
			schema["format"] = "email"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema["type"] = "integer"
		// Parse jsonschema tag for min/max
		if strings.Contains(jsonschemaTag, "minimum=") {
			// Extract minimum value
		}
		if strings.Contains(jsonschemaTag, "maximum=") {
			// Extract maximum value
		}
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		// Handle array items
		if fieldType.Elem().Kind() == reflect.String {
			schema["items"] = map[string]interface{}{"type": "string"}
		}
	case reflect.Struct:
		schema["type"] = "object"
		// For nested structs, we could recursively build the schema
	}

	// Parse description from jsonschema tag
	if strings.Contains(jsonschemaTag, "description=") {
		// Extract description
	}

	return schema
}
