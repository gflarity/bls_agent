package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gflarity/bls_agent/pkg/llm"
	"github.com/joho/godotenv"
)

// Person is a sample struct to demonstrate jsonschema generation
type Person struct {
	Name      string   `json:"name" jsonschema:"description=The name of the person"`
	Age       int      `json:"age" jsonschema:"description=The age of the person,minimum=0,maximum=150"`
	IsStudent bool     `json:"is_student" jsonschema:"description=Whether the person is a student"`
	Courses   []string `json:"courses" jsonschema:"description=List of courses the person is taking"`
	Email     string   `json:"email" jsonschema:"description=Email address,format=email"`
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		// It's okay if .env file doesn't exist, just log it
		fmt.Println("ℹ️  No .env file found, using system environment variables")
	}

	fmt.Println("=== LLM Package with JSON Schema Demo ===")

	// Check if API key is set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("OPENAI_API_KEY is not set")
	}

	// Generate JSON schema from the Person struct
	fmt.Println("1. Generating JSON Schema from Go Struct...")
	schema, err := llm.GenerateSchemaFromType(Person{})
	if err != nil {
		panic(fmt.Errorf("failed to generate schema from type: %w", err))
	}

	// Pretty print the generated schema
	schemaBytes, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Printf("Generated Schema:\n%s\n\n", string(schemaBytes))

	// Marshal the schema map to a JSON string for CompleteWithSchema
	schemaStr := string(schemaBytes)

	fmt.Println("\n4. Making API Call...")

	// Get additional environment variables
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		panic("OPENAI_BASE_URL is not set")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		panic("OPENAI_MODEL is not set")
	}

	fmt.Printf("Using Base URL: %s\n", baseURL)
	fmt.Printf("Using Model: %s\n", model)

	cont, reas, err := llm.CompleteWithSchema(
		context.Background(),
		apiKey,
		baseURL,
		schemaStr, // Using the JSON schema string
		"You are a helpful assistant that extracts information and returns it in JSON format.",
		"Extract the following information: Jane Smith is 25 years old, is a student, and is taking 'Computer Science' and 'Physics'.",
		model,
	)

	if err != nil {
		fmt.Printf("❌ CompleteWithSchema API call failed: %v\n", err)
	} else {
		// Try to unmarshal the response to validate it matches our schema
		var person Person
		if unmarshalErr := json.Unmarshal([]byte(cont), &person); unmarshalErr != nil {
			fmt.Printf("❌ API call succeeded but response is not valid JSON for Person struct: %v\n", unmarshalErr)
		} else {
			fmt.Printf("✅ CompleteWithSchema API call successful!\n")
			fmt.Printf("✅ Response successfully unmarshaled into Person struct:\n")
		}
		fmt.Printf("Content: %s\n", cont)
		if reas != "" {
			fmt.Printf("Reasoning: %s\n", reas)
		}
	}
}
