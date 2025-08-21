package bls

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/apognu/gocal"
)

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
func GetAllEvents() ([]gocal.Event, error) {
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
	if len(bodyStr) == 0 {
		return nil, fmt.Errorf("empty response from calendar URL")
	}

	log.Printf("Calendar response preview: %s", bodyStr[:min(len(bodyStr), 200)])

	if !strings.Contains(bodyStr, "BEGIN:VCALENDAR") {
		return nil, fmt.Errorf("response does not appear to be valid ICS calendar data")
	}

	var cleanedLines []string
	for _, line := range strings.Split(bodyStr, "\n") {
		if strings.TrimSpace(line) != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	cleanedBody := strings.Join(cleanedLines, "\n")

	var processedLines []string
	lines := strings.Split(cleanedBody, "\n")
	inEvent := false
	hasDtstamp := false
	for _, line := range lines {
		processedLines = append(processedLines, line)
		if strings.TrimSpace(line) == "BEGIN:VEVENT" {
			inEvent = true
			hasDtstamp = false
		} else if strings.TrimSpace(line) == "END:VEVENT" {
			inEvent = false
		} else if inEvent && strings.HasPrefix(strings.TrimSpace(line), "DTSTAMP:") {
			hasDtstamp = true
		} else if inEvent && strings.HasPrefix(strings.TrimSpace(line), "SUMMARY:") && !hasDtstamp {
			dtstamp := fmt.Sprintf("DTSTAMP:%s", time.Now().UTC().Format("20060102T150405Z"))
			processedLines = append(processedLines[:len(processedLines)-1], dtstamp, line)
			hasDtstamp = true
		}
	}
	processedBody := strings.Join(processedLines, "\n")

	parser := gocal.NewParser(strings.NewReader(processedBody))
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in gocal parsing: %v", r)
		}
	}()

	err = parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse calendar: %w", err)
	}

	if len(parser.Events) == 0 {
		log.Println("No events found in calendar")
		return []gocal.Event{}, nil
	}

	return parser.Events, nil
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
func FindEvents(mins float64) ([]gocal.Event, error) {
	allEvents, err := GetAllEvents()
	if err != nil {
		return nil, err
	}

	var recentEvents []gocal.Event
	for _, event := range allEvents {
		if event.Start == nil {
			continue
		}
		age := AgeInMins(*event.Start)
		if age < mins && age > 0 {
			log.Printf("Found event within the last %.f minutes: %s", mins, event.Summary)
			recentEvents = append(recentEvents, event)
		}
	}
	return recentEvents, nil
}

// FindUpcomingEvents finds events scheduled within the next `mins` minutes.
func FindUpcomingEvents(mins float64) ([]gocal.Event, error) {
	allEvents, err := GetAllEvents()
	if err != nil {
		return nil, err
	}

	var upcomingEvents []gocal.Event
	now := time.Now()
	for _, event := range allEvents {
		if event.Start == nil {
			continue
		}
		timeUntil := event.Start.Sub(now).Minutes()
		if timeUntil >= 0 && timeUntil <= mins {
			log.Printf("Found upcoming event in %.f minutes: %s", timeUntil, event.Summary)
			upcomingEvents = append(upcomingEvents, event)
		}
	}
	return upcomingEvents, nil
}

// FetchReleaseHTML fetches the HTML for the release of an event.
func FetchReleaseHTML(event gocal.Event) (string, error) {
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
