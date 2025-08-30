package bls

import (
	"strings"
	"testing"
	"time"
)

// === UNIT TESTS ===
// These tests are fast and don't rely on network access.

func TestExtractSummary(t *testing.T) {
	// t.Parallel() // You can run tests in parallel if they don't share state.

	// Define test cases in a table-driven format, which is idiomatic in Go.
	testCases := []struct {
		name        string
		htmlInput   string
		want        string
		expectErr   bool
		expectedErr string
	}{
		{
			name:      "Happy Path - Summary with separator",
			htmlInput: `<html><body><pre>This is the summary.          __________     Some other text.</pre></body></html>`,
			want:      "This is the summary.          ",
			expectErr: false,
		},
		{
			name:      "Happy Path - Summary without separator",
			htmlInput: `<html><body><pre>This is a complete summary without a line.</pre></body></html>`,
			want:      "This is a complete summary without a line.",
			expectErr: false,
		},
		{
			name:        "Error - No <pre> tag",
			htmlInput:   `<html><body><p>There is no preformatted text here.</p></body></html>`,
			want:        "",
			expectErr:   true,
			expectedErr: "no <pre> tag found",
		},
		{
			name:        "Error - Empty input",
			htmlInput:   "",
			want:        "",
			expectErr:   true,
			expectedErr: "no <pre> tag found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractSummary(tc.htmlInput)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected an error but got nil")
				}
				if err != nil && !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("Expected error containing '%s', but got: %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if got != tc.want {
					t.Errorf("ExtractSummary() = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestAgeInMins(t *testing.T) {
	// t.Parallel()

	// Create a time object that represents 15 minutes ago
	pastTime := time.Now().Add(-15 * time.Minute)
	age := AgeInMins(pastTime)

	// We check if the age is within a reasonable range to account for execution time.
	if age < 14.99 || age > 15.01 {
		t.Errorf("Expected age to be ~15.0 minutes, but got %f", age)
	}
}

// === INTEGRATION TESTS ===
// These tests perform live network requests. They are slower and can be brittle.
// They are skipped by default unless the -short flag is omitted.

func TestGetAllEvents(t *testing.T) {
	events, err := GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents() returned an error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("GetAllEvents() returned 0 events, expected at least one.")
	}

	t.Logf("Successfully fetched %d events.", len(events))
}

func TestFetchReleaseHTML(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}

	// Create a test event for a stable, well-known release
	startTime := time.Time{}
	cpiEvent := Event{
		Summary: "Consumer Price Index",
		Start:   &startTime, // Start time doesn't matter for this test
	}

	t.Run("Happy Path - Fetch known release", func(t *testing.T) {
		html, err := FetchReleaseHTML(cpiEvent)
		if err != nil {
			t.Fatalf("FetchReleaseHTML() returned an error for a valid event: %v", err)
		}
		if len(html) < 100 {
			t.Errorf("Expected a substantial HTML document, but got a very short string (len %d)", len(html))
		}
		if !strings.Contains(html, "Consumer Price Index") {
			t.Error("Expected HTML to contain the event title 'Consumer Price Index'.")
		}
	})

	t.Run("Error - Event not in map", func(t *testing.T) {
		unknownEvent := Event{
			Summary: "An Imaginary Economic Indicator",
		}
		_, err := FetchReleaseHTML(unknownEvent)
		if err == nil {
			t.Fatal("Expected an error for an unknown event, but got nil.")
		}
		expectedErr := "no mapping for event"
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf("Expected error to contain '%s', but got: %v", expectedErr, err)
		}
	})
}
