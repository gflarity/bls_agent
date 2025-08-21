package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

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

// main function to demonstrate the usage of CompleteWithSchema.
func main() {
	// It's best practice to load the API key from an environment variable.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY environment variable not set.")
	}

	// Define a sample schema for the request.
	sampleSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":       map[string]interface{}{"type": "string", "description": "The name of the person."},
			"age":        map[string]interface{}{"type": "integer", "description": "The age of the person."},
			"is_student": map[string]interface{}{"type": "boolean", "description": "Whether the person is a student."},
			"courses":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"name", "age", "is_student"},
	}

	// Call the function with sample data.
	content, reasoning, err := CompleteWithSchema(
		context.Background(),
		apiKey,
		"https://api.openai.com/v1", // <-- IMPORTANT: Change this to your CentML or other custom base URL if needed.
		sampleSchema,
		"You are a helpful assistant that extracts information and returns it in JSON format.",
		"Extract the following information: John Doe is 30 years old, not a student, and is taking 'History' and 'Math'.",
		openai.ChatModelGPT4o, // Using a standard model constant from the official library.
	)

	if err != nil {
		log.Fatalf("Function call failed: %v", err)
	}

	// Print the results.
	fmt.Println("--- LLM Response ---")
	fmt.Println(content)
	fmt.Println("\n--- Reasoning ---")
	fmt.Printf("'%s' (Note: This is often empty as it's a non-standard field)\n", reasoning)

	// You can also unmarshal the JSON string into a Go struct for further processing.
	type Person struct {
		Name      string   `json:"name"`
		Age       int      `json:"age"`
		IsStudent bool     `json:"is_student"`
		Courses   []string `json:"courses"`
	}

	var personData Person
	if err := json.Unmarshal([]byte(content), &personData); err != nil {
		log.Fatalf("Failed to unmarshal response content: %v", err)
	}

	fmt.Println("\n--- Parsed Data ---")
	fmt.Printf("Name: %s\n", personData.Name)
	fmt.Printf("Age: %d\n", personData.Age)
	fmt.Printf("Is Student: %t\n", personData.IsStudent)
	fmt.Printf("Courses: %v\n", personData.Courses)
}
