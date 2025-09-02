package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gflarity/bls_agent/pkg/arxiv"
)

func main() {
	// Test the new API-based approach
	fmt.Println("Testing ArXiv API-based paper fetching...")

	// Test with the exact date from your example
	targetDate := time.Date(2025, 8, 28, 0, 0, 0, 0, time.UTC)

	fmt.Printf("Fetching papers submitted on %s using ArXiv API...\n", targetDate.Format("2006-01-02"))

	// Test the ID-only function
	ids, err := arxiv.GetArxivIdsForDateAPI(targetDate)
	if err != nil {
		log.Fatalf("Error fetching ArXiv IDs: %v", err)
	}

	fmt.Printf("Found %d papers submitted on %s:\n", len(ids), targetDate.Format("2006-01-02"))
	for i, id := range ids {
		fmt.Printf("%d. %s\n", i+1, id)
		if i >= 4 { // Limit output to first 5 papers
			fmt.Printf("... and %d more\n", len(ids)-5)
			break
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 50))

	// Test the full metadata function
	fmt.Println("Fetching full paper metadata...")
	fmt.Println("Note: ArXiv requires 3-second delays between API requests - this may take a moment for large result sets...")
	entries, err := arxiv.GetArxivPapersBySubmissionDate(targetDate, "cs.AI")
	if err != nil {
		log.Fatalf("Error fetching paper metadata: %v", err)
	}

	fmt.Printf("Found %d papers with full metadata:\n", len(entries))
	for i, entry := range entries {
		fmt.Printf("\n%d. %s\n", i+1, entry.ArxivID())
		fmt.Printf("   Title: %s\n", entry.CleanTitle())
		fmt.Printf("   Published: %s\n", entry.Published.Format("2006-01-02 15:04:05"))

		// Show authors
		var authors []string
		for _, author := range entry.Authors {
			authors = append(authors, author.Name)
		}
		if len(authors) > 0 {
			fmt.Printf("   Authors: %s\n", strings.Join(authors, ", "))
		}

		// Show first few words of abstract
		abstract := entry.CleanSummary()
		if len(abstract) > 100 {
			abstract = abstract[:100] + "..."
		}
		fmt.Printf("   Abstract: %s\n", abstract)

		/*
			if i >= 2 { // Limit to first 3 papers for detailed output
				fmt.Printf("\n... and %d more papers\n", len(entries)-3)
				break
			}
		*/
	}
}
