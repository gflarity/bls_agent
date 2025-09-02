package main

import (
	"log"
	"time"

	"github.com/gflarity/bls_agent/pkg/arxiv"
)

func main() {
	// Define the target date. Let's look for papers 3 days.
	targetDate := time.Now().AddDate(0, 0, -4)
	log.Printf("Searching for papers from: %s\n", targetDate.Format("2006-01-02"))

	// 1. Get all paper IDs for the target date.
	ids, err := arxiv.GetArxivIdsForDate(targetDate)
	if err != nil {
		log.Fatalf("Failed to get arxiv IDs: %v", err)
	}

	if len(ids) == 0 {
		log.Println("No papers found for this date.")
		return
	}
	log.Printf("Found %d paper(s) for the target date.\n", len(ids))

	// 3. Let's process the first paper as an example.
	if len(ids) > 0 {
		firstPaper := ids[0]
		log.Printf("\n--- Processing paper: %s ---\n", firstPaper)

		// Lazily fetch the abstract.
		abstract, err := arxiv.GetArxivAbstract(firstPaper)
		if err != nil {
			log.Fatalf("Failed to get abstract: %v", err)
		}
		log.Printf("Abstract: %s...\n", abstract)

		paperText, err := arxiv.ExtractPaperText(firstPaper)
		if err != nil {
			log.Fatalf("Failed to get paper text: %v", err)
		}
		log.Printf("Paper text: %s...\n", paperText)
	}

}
