// Package arxiv provides tools for fetching and processing academic papers from arxiv.org.
package arxiv

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ledongthuc/pdf"
)

// httpClient is a shared HTTP client with a timeout for all requests.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// whitespaceRegex is used to clean up abstract text.
var whitespaceRegex = regexp.MustCompile(`\s+`)

// ArxivAPIResponse represents the XML response structure from ArXiv API
type ArxivAPIResponse struct {
	XMLName      xml.Name   `xml:"feed"`
	TotalResults int        `xml:"totalResults"`
	StartIndex   int        `xml:"startIndex"`
	ItemsPerPage int        `xml:"itemsPerPage"`
	Entries      []APIEntry `xml:"entry"`
}

// APIEntry represents a single paper entry in the ArXiv API response
type APIEntry struct {
	ID         string        `xml:"id"`
	Published  time.Time     `xml:"published"`
	Updated    time.Time     `xml:"updated"`
	Title      string        `xml:"title"`
	Summary    string        `xml:"summary"`
	Authors    []APIAuthor   `xml:"author"`
	Categories []APICategory `xml:"category"`
}

// APIAuthor represents an author in the ArXiv API response
type APIAuthor struct {
	Name string `xml:"name"`
}

// APICategory represents a category in the ArXiv API response
type APICategory struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

// ArxivID extracts the ArXiv ID from the full ID URL
func (e *APIEntry) ArxivID() string {
	// ID comes as "http://arxiv.org/abs/2508.21263v1"
	parts := strings.Split(e.ID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// CleanTitle removes extra whitespace from the title
func (e *APIEntry) CleanTitle() string {
	return whitespaceRegex.ReplaceAllString(strings.TrimSpace(e.Title), " ")
}

// CleanSummary removes extra whitespace from the summary/abstract
func (e *APIEntry) CleanSummary() string {
	return whitespaceRegex.ReplaceAllString(strings.TrimSpace(e.Summary), " ")
}

// Paper represents an academic paper with metadata and content.
// It uses lazy-loading to fetch the abstract and full text on demand.
type Paper struct {
	ArxivID string `json:"arxivId"`
	// Exported for JSON marshaling, but access is intended via methods.
	InternalAbstract string `json:"_abstract"`
	InternalText     string `json:"_text"`
	// mu protects the lazy-loading logic for concurrent access.
	mu sync.Mutex
}

// GetArxivIdsForDate scrapes the Arxiv "recent" page to find all paper IDs
// published on a specific target date for the cs.AI category.
func GetArxivIdsForDate(targetDate time.Time) ([]string, error) {
	const url = "https://arxiv.org/list/cs.AI/recent?skip=0&show=2000"
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error fetching paper list: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Arxiv headers include the date like "Announcements for Fri, 30 Aug 2025"
	// We'll search for the date part, e.g., "30 Aug 2025".
	dateStr := targetDate.Format("2 Jan 2006")
	var dateHeader *goquery.Selection

	doc.Find("dl#articles h3").EachWithBreak(func(i int, h3 *goquery.Selection) bool {
		if strings.Contains(h3.Text(), dateStr) {
			dateHeader = h3
			return false // Stop searching
		}
		return true // Continue
	})

	if dateHeader == nil {
		return []string{}, nil // No papers found for this date, not an error.
	}

	var arxivIds []string
	// Traverse the DOM siblings after the header until we hit the next header.
	currentNode := dateHeader.Next()
	for currentNode.Length() > 0 && !currentNode.Is("h3") {
		// Paper entries are in <dt> tags.
		if currentNode.Is("dt") {
			link := currentNode.Find(`a[href^="/abs/"]`)
			if href, exists := link.Attr("href"); exists {
				id := strings.TrimPrefix(href, "/abs/")
				if id != "" {
					arxivIds = append(arxivIds, id)
				}
			}
		}
		currentNode = currentNode.Next()
	}

	return arxivIds, nil
}

// GetArxivAbstract fetches and cleans the abstract text from a paper's abstract page.
func GetArxivAbstract(arxivId string) (string, error) {
	url := fmt.Sprintf("https://arxiv.org/abs/%s", arxivId)
	res, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error fetching abstract: %s", res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	abstract := doc.Find("blockquote.abstract").Text()
	if abstract == "" {
		return "", fmt.Errorf("abstract blockquote not found in document for id %s", arxivId)
	}

	// Clean the abstract text by removing the "Abstract: " prefix and normalizing whitespace.
	cleaned := strings.TrimPrefix(abstract, "Abstract: ")
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned), nil
}

// ExtractPaperText fetches a paper's PDF and extracts its plain text content.
func ExtractPaperText(arxivId string) (string, error) {
	url := fmt.Sprintf("https://arxiv.org/pdf/%s", arxivId)
	res, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch pdf: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	reader := bytes.NewReader(body)

	pdfReader, err := pdf.NewReader(reader, int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("failed to create pdf reader: %w", err)
	}

	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// GetTextByRow provides text chunks in order, which allows us to reconstruct lines with spaces.
		rows, err := page.GetTextByRow()
		if err != nil {
			return "", fmt.Errorf("failed to get text by row on page %d: %w", i, err)
		}

		for _, row := range rows {
			var words []string
			for _, word := range row.Content {
				words = append(words, word.S)
			}
			// Join the words in the row with spaces and add a newline.
			textBuilder.WriteString(strings.Join(words, " "))
			textBuilder.WriteString("\n")
		}
	}

	return textBuilder.String(), nil
}

// GetArxivPapersBySubmissionDate fetches papers submitted on a specific date using the ArXiv API.
// It supports pagination and returns all papers for the given date and category.
//
// IMPORTANT: This function respects ArXiv's Terms of Use by waiting 3 seconds between API requests
// when pagination is required. This means fetching large result sets may take several minutes.
func GetArxivPapersBySubmissionDate(targetDate time.Time, category string) ([]APIEntry, error) {
	if category == "" {
		category = "cs.AI" // Default to AI papers
	}

	// Format date for ArXiv API (YYYYMMDDHHMM format)
	// We want the full day, so we search from 00:00 to 23:59
	startDate := targetDate.Format("200601021504")                               // YYYYMMDDHHMM - start of day (00:00)
	endDate := targetDate.Add(24*time.Hour - time.Minute).Format("200601021504") // End of day (23:59)

	var allEntries []APIEntry
	start := 0
	maxResults := 100 // ArXiv API limit per request

	for {
		entries, totalResults, err := fetchArxivPage(category, startDate, endDate, start, maxResults)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch ArXiv page: %w", err)
		}

		allEntries = append(allEntries, entries...)

		// Check if we've retrieved all results
		if start+len(entries) >= totalResults || len(entries) == 0 {
			break
		}

		start += maxResults
		// Note: fetchArxivPage will automatically wait 3 seconds before the next request
		// as required by ArXiv's Terms of Use
	}

	return allEntries, nil
}

// fetchArxivPage fetches a single page of results from the ArXiv API
func fetchArxivPage(category, startDate, endDate string, start, maxResults int) ([]APIEntry, int, error) {
	// ArXiv Terms of Use: Wait at least 3 seconds between requests to avoid being blocked
	// Only sleep if this is not the first request (start > 0)
	if start > 0 {
		time.Sleep(3 * time.Second)
	}

	// Build the search query
	searchQuery := fmt.Sprintf("cat:%s AND submittedDate:[%s TO %s]", category, startDate, endDate)

	// Build the API URL
	params := url.Values{}
	params.Set("search_query", searchQuery)
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")
	params.Set("start", strconv.Itoa(start))
	params.Set("max_results", strconv.Itoa(maxResults))

	apiURL := "https://export.arxiv.org/api/query?" + params.Encode()

	// Make the HTTP request
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Read and parse the XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse ArxivAPIResponse
	if err := xml.Unmarshal(body, &apiResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to parse XML response: %w", err)
	}

	return apiResponse.Entries, apiResponse.TotalResults, nil
}

// GetArxivIdsForDateAPI is an API-based alternative to GetArxivIdsForDate
// that uses the ArXiv API instead of scraping HTML pages.
func GetArxivIdsForDateAPI(targetDate time.Time) ([]string, error) {
	entries, err := GetArxivPapersBySubmissionDate(targetDate, "cs.AI")
	if err != nil {
		return nil, err
	}

	var arxivIds []string
	for _, entry := range entries {
		if id := entry.ArxivID(); id != "" {
			arxivIds = append(arxivIds, id)
		}
	}

	return arxivIds, nil
}
