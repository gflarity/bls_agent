package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gflarity/bls_agent/pkg/bls"
)

func main() {
	fmt.Println("BLS Events Tester")
	fmt.Println("==================")

	// Get all events and filter for the last week
	fmt.Printf("\nFetching all events from the calendar...\n")
	allEvents, err := bls.GetAllEvents()
	if err != nil {
		log.Printf("Error getting all events: %v", err)
		return
	}

	fmt.Printf("Total events in calendar: %d\n", len(allEvents))

	// Filter events from the last week (7 days) - only past events
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	var recentEvents []bls.Event

	for _, event := range allEvents {
		if event.Start != nil && event.Start.After(oneWeekAgo) && event.Start.Before(time.Now()) {
			recentEvents = append(recentEvents, event)
		}
	}

	fmt.Printf("\nEvents in the last week: %d\n", len(recentEvents))

	if len(recentEvents) > 0 {
		fmt.Println("\nRecent events:")
		for i, event := range recentEvents {
			fmt.Printf("  %d. %s\n", i+1, event.Summary)
			if event.Start != nil {
				timeUntil := event.Start.Sub(time.Now()).Minutes()
				if timeUntil > 0 {
					fmt.Printf("     Starts: %s (in %.1f minutes)\n",
						event.Start.Format("2006-01-02 15:04:05"), timeUntil)
				} else {
					age := bls.AgeInMins(*event.Start)
					fmt.Printf("     Started: %s (%.1f minutes ago)\n",
						event.Start.Format("2006-01-02 15:04:05"), age)
				}
			}
		}
	} else {
		fmt.Println("No events found in the last week.")
	}

	fmt.Println("\nTest completed!")
}
