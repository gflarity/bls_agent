package bls

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/PuloV/ics-golang"
)

// Event represents a calendar event with the fields we need
type Event struct {
	Summary string
	Start   *time.Time
	End     *time.Time
	UID     string
}

// eventMappings holds the mapping of event summaries to their news release URLs.
var eventMappings = map[string]string{
	"State Job Openings and Labor Turnover":                   "https://www.bls.gov/news.release/jltst.nr0.htm",
	"Employer Costs for Employee Compensation":                "https://www.bls.gov/news.release/ecec.nr0.htm",
	"Job Openings and Labor Turnover Survey":                  "https://www.bls.gov/news.release/jolts.nr0.htm",
	"Metropolitan Area Employment and Unemployment (Monthly)": "https://www.bls.gov/news.release/metro.nr0.htm",
	"Employment Situation":                                    "https://www.bls.gov/news.release/empsit.nr0.htm",
	"Real Earnings":                                           "https://www.bls.gov/news.release/realer.nr0.htm",
	"Consumer Price Index":                                    "https://www.bls.gov/news.release/cpi.nr0.htm",
	"Producer Price Index":                                    "https://www.bls.gov/news.release/ppi.nr0.htm",
	"U.S. Import and Export Price Indexes":                    "https://www.bls.gov/news.release/ximpim.nr0.htm",
	"Usual Weekly Earnings of Wage and Salary Workers":        "https://www.bls.gov/news.release/wkyeng.nr0.htm",
	"State Employment and Unemployment (Monthly)":             "https://www.bls.gov/news.release/laus.htm",
	"Union Membership (Annual)":                               "https://www.bls.gov/news.release/union2.nr0.htm",
	"Quarterly Data Series on Business Employment Dynamics":   "https://www.bls.gov/news.release/cewbd.nr0.htm",
	"Employment Cost Index":                                   "https://www.bls.gov/news.release/eci.nr0.htm",
	"Productivity and Costs":                                  "https://www.bls.gov/news.release/prod2.nr0.htm",
	"Occupational Requirements in the United States":          "https://www.bls.gov/news.release/ors.nr0.htm",
	"Major Work Stoppages (Annual)":                           "https://www.bls.gov/news.release/wkstp.nr0.htm",
	"County Employment and Wages":                             "https://www.bls.gov/news.release/cewqtr.nr0.htm",
	"Persons with a Disability: Labor Force Characteristics":  "https://www.bls.gov/news.release/disabl.nr0.htm",
	"State Unemployment (Annual)":                             "https://www.bls.gov/news.release/disabl.nr0.htm",
	"Employment Situation of Veterans":                        "https://www.bls.gov/news.release/vet.nr0.htm",
	"Total Factor Productivity":                               "https://www.bls.gov/news.release/prod3.nr0.htm",
	"Labor Market Experience, Education, Partner Status, and Health for those Born 1980-1984": "https://www.bls.gov/news.release/nlsyth.nr0.htm",
	"Occupational Employment and Wages":                                                            "https://www.bls.gov/news.release/ocwage.nr0.htm",
	"College Enrollment and Work Activity of High School Graduates":                                "https://www.bls.gov/news.release/hsgec.nr0.htm",
	"Employment Characteristics of Families":                                                       "https://www.bls.gov/news.release/famee.nr0.htm",
	"Productivity and Costs by Industry: Manufacturing and Mining Industries":                      "https://www.bls.gov/news.release/prin.nr0.htm",
	"Labor Force Characteristics of Foreign-born Workers":                                          "https://www.bls.gov/news.release/forbrn.nr0.htm",
	"Productivity and Costs by Industry: Wholesale Trade and Retail Trade":                         "https://www.bls.gov/news.release/prin1.nr0.htm",
	"Productivity by State":                                                                        "https://www.bls.gov/news.release/prin4.nr0.htm",
	"American Time Use Survey":                                                                     "https://www.bls.gov/news.release/atus.nr0.htm",
	"Productivity and Costs by Industry: Selected Service-Providing Industries":                    "https://www.bls.gov/news.release/prin2.nr0.htm",
	"Summer Youth Labor Force":                                                                     "https://www.bls.gov/news.release/youth.nr0.htm",
	"Employment Projections and Occupational Outlook Handbook":                                     "https://www.bls.gov/news.release/ecopro.nr0.htm",
	"Worker Displacement":                                                                          "https://www.bls.gov/news.release/disp.nr0.htm",
	"Employee Benefits in the United States":                                                       "https://www.bls.gov/news.release/ebs2.nr0.htm",
	"Consumer Expenditures":                                                                        "https://www.bls.gov/news.release/cesan.nr0.htm",
	"Employee Tenure":                                                                              "https://www.bls.gov/news.release/tenure.nr0.htm",
	"Employer-Reported Workplace Injuries and Illnesses (Annual)":                                  "https://www.bls.gov/news.release/osh.nr0.htm",
	"Contingent and Alternative Employment Arrangements":                                           "https://www.bls.gov/news.release/conemp.nr0.htm",
	"Total Factor Productivity for Major Industries":                                               "https://www.bls.gov/news.release/prod5.nr0.htm",
	"Work Experience of the Population (Annual)":                                                   "https://www.bls.gov/news.release/work.nr0.htm",
	"Census of Fatal Occupational Injuries":                                                        "https://www.bls.gov/news.release/cfoi.nr0.htm",
	"Number of Jobs, Labor Market Experience, Marital Status, and Health for those Born 1957-1964": "https://www.bls.gov/news.release/nlsoy.nr0.htm",
	"Unpaid Eldercare in the United States":                                                        "https://www.bls.gov/news.release/elcare.nr0.htm",
}

// GetAllEvents fetches the BLS calendar and returns all events.
// This version has been corrected to use the ics-golang library more idiomatically.
func GetAllEvents() ([]Event, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.bls.gov/schedule/news_release/bls.ics", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/calendar,text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from calendar URL: %s", resp.Status)
	}

	var bodyReader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		bodyReader = gzipReader
	}

	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	bodyStr := string(bodyBytes)

	fmt.Println("bodyStr length: ", len(bodyStr))
	sha := sha256.New()
	sha.Write([]byte(bodyStr))
	fmt.Println("sha: ", hex.EncodeToString(sha.Sum(nil)))

	if !strings.Contains(bodyStr, "BEGIN:VCALENDAR") {
		return nil, fmt.Errorf("response does not appear to be valid ICS calendar data")
	}

	// --- FIX: Replace non-standard timezone with IANA standard ---
	// The BLS calendar uses "US-Eastern", which Go's time package cannot parse.
	// We replace it with the official IANA name "America/New_York".
	// Replace all occurrences of US-Eastern timezone references
	bodyStr = strings.ReplaceAll(bodyStr, "TZID=US-Eastern", "TZID=America/New_York")
	bodyStr = strings.ReplaceAll(bodyStr, "X-WR-TIMEZONE:US-Eastern", "X-WR-TIMEZONE:America/New_York")

	// --- REVISED PARSING LOGIC ---
	parser := ics.New()
	parser.Load(bodyStr)

	// The parser works asynchronously. We need to read events from its output channel.
	outputChan := parser.GetOutputChan()

	var icsEvents []*ics.Event

	// Use a goroutine to collect events with a timeout to prevent hanging
	done := make(chan bool)
	go func() {
		for event := range outputChan {
			icsEvents = append(icsEvents, event)
		}
		done <- true
	}()

	// Wait for either completion or timeout
	// TODO this is pretty sketchy, need to improve it, maybe use gocal again
	// might be related to the cleaning up of the bodyStr
	select {
	case <-done:
		// Events collected successfully
	case <-time.After(2 * time.Second):
		// Timeout after 10 seconds
		log.Println("Warning: Parser timeout reached, proceeding with collected events")
	}

	// After waiting, check if the parser encountered any errors.
	parserErrors, _ := parser.GetErrors()
	if len(parserErrors) > 0 {
		// Log the errors but don't necessarily fail if some events were parsed.
		log.Printf("Warning: encountered errors while parsing ICS data: %v", parserErrors)
	}

	if len(icsEvents) == 0 {
		log.Println("No events found in calendar")
		return []Event{}, nil
	}

	// Convert from the library's event type to our custom Event type.
	var events []Event
	for _, icsEvent := range icsEvents {
		if icsEvent == nil {
			continue
		}

		var startTime, endTime *time.Time
		start := icsEvent.GetStart()
		if !start.IsZero() {
			startTime = &start
		}
		end := icsEvent.GetEnd()
		if !end.IsZero() {
			endTime = &end
		}

		ourEvent := Event{
			Summary: icsEvent.GetSummary(),
			UID:     icsEvent.GetID(),
			Start:   startTime,
			End:     endTime,
		}
		events = append(events, ourEvent)
	}

	return events, nil
}

// min is a helper function to find the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AgeInMins calculates the age of a date in minutes.
func AgeInMins(t time.Time) float64 {
	return time.Since(t).Minutes()
}

// FindEvents finds events that happened within the last `mins` minutes.
// This version only returns past events, not future ones.
func FindEvents(mins float64) ([]Event, error) {
	allEvents, err := GetAllEvents()
	fmt.Println("allEvents: ", len(allEvents))
	if err != nil {
		return nil, err
	}

	// Use precise duration arithmetic for exact minute precision
	cutoffTime := time.Now().Add(-time.Duration(mins) * time.Minute)
	var recentEvents []Event

	for _, event := range allEvents {
		if event.Start != nil && event.Start.After(cutoffTime) && event.Start.Before(time.Now()) {
			recentEvents = append(recentEvents, event)
		}
	}

	return recentEvents, nil
}

// FetchReleaseHTML fetches the HTML for the release of an event.
func FetchReleaseHTML(event Event) (string, error) {
	summary := strings.TrimSpace(event.Summary)
	url, ok := eventMappings[summary]
	if !ok {
		return "", fmt.Errorf("no mapping for event: %s", summary)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch HTML from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		if event.Start != nil && event.Start.After(time.Now()) {
			return "", fmt.Errorf("release not yet available (future event scheduled for %s)", event.Start.Format("2006-01-02 15:04"))
		}
		return "", fmt.Errorf("access forbidden - release may not be publicly available")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status from %s: %s", url, resp.Status)
	}

	var bodyReader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		bodyReader = gzipReader
	}

	html, err := io.ReadAll(bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(html), nil
}

// ExtractSummary extracts the summary text from the release HTML.
func ExtractSummary(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse document: %w", err)
	}

	pre := doc.Find("pre")
	if pre.Length() == 0 {
		return "", fmt.Errorf("no <pre> tag found in document")
	}

	summary := pre.Text()
	if idx := strings.Index(summary, "__________"); idx != -1 {
		return summary[:idx], nil
	}

	return summary, nil
}
