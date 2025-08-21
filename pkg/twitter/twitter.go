// main.go
package twitter

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/g8rswimmer/go-twitter/v2"
)

// twitterClient encapsulates the authenticated Twitter API v2 client.
type twitterClient struct {
	*twitter.Client
}

// authorizer is a dummy struct to satisfy the go-twitter client interface.
// Authorization is already handled by the underlying oauth1 http.Client.
type authorizer struct{}

func (a *authorizer) Add(req *http.Request) {
	// This is a no-op because the http.Client from the oauth1 library
	// automatically handles adding the Authorization header to requests.
}

// NewClient configures and returns a new Twitter client using credentials
// from environment variables.
func NewClient() (*twitterClient, error) {
	// Read credentials from environment variables
	consumerKey := os.Getenv("X_API_KEY")
	consumerSecret := os.Getenv("X_API_SECRET")
	accessToken := os.Getenv("X_ACCESS_TOKEN")
	accessTokenSecret := os.Getenv("X_ACCESS_TOKEN_SECRET")

	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessTokenSecret == "" {
		return nil, fmt.Errorf("missing required environment variables: X_API_KEY, X_API_SECRET, X_ACCESS_TOKEN, X_ACCESS_TOKEN_SECRET")
	}

	return NewClientWithCredentials(consumerKey, consumerSecret, accessToken, accessTokenSecret)
}

// NewClientWithCredentials configures and returns a new Twitter client using
// the provided credentials.
func NewClientWithCredentials(consumerKey, consumerSecret, accessToken, accessTokenSecret string) (*twitterClient, error) {
	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessTokenSecret == "" {
		return nil, fmt.Errorf("all credentials must be provided")
	}

	// Create an OAuth1 config and token
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessTokenSecret)

	// Create an http.Client that will automatically sign requests
	httpClient := config.Client(context.Background(), token)

	// Create the go-twitter v2 client
	client := &twitterClient{
		Client: &twitter.Client{
			Authorizer: &authorizer{},
			Client:     httpClient,
			Host:       "https://api.twitter.com",
		},
	}

	return client, nil
}

// PostTweet posts a single tweet. It can optionally reply to another tweet.
// It returns the new tweet's ID on success.
func (c *twitterClient) PostTweet(text string, replyToID string) (string, error) {
	req := twitter.CreateTweetRequest{
		Text: text,
	}

	// If a replyToID is provided, structure the request as a reply
	if replyToID != "" {
		req.Reply = &twitter.CreateTweetReply{
			InReplyToTweetID: replyToID,
		}
	}

	fmt.Printf("Posting tweet: \"%s\"\n", text)
	res, err := c.CreateTweet(context.Background(), req)
	if err != nil {
		// The library might wrap the original error, so we print the whole chain.
		return "", fmt.Errorf("error posting tweet: %w", err)
	}

	// Check if the tweet was created successfully
	if res.Tweet == nil {
		return "", fmt.Errorf("twitter API returned an empty tweet object")
	}

	fmt.Printf("✅ Successfully posted tweet ID: %s\n", res.Tweet.ID)
	return res.Tweet.ID, nil
}

// PostTweetThread posts a slice of strings as a threaded tweet conversation.
func (c *twitterClient) PostTweetThread(texts []string) error {
	if len(texts) == 0 {
		return fmt.Errorf("no tweets provided in the thread")
	}

	var previousTweetID string
	for i, text := range texts {
		tweetID, err := c.PostTweet(text, previousTweetID)
		if err != nil {
			return fmt.Errorf("failed to post tweet #%d in thread: %w", i+1, err)
		}
		previousTweetID = tweetID

		// Pause between tweets to avoid rate limiting, but not after the last one
		if i < len(texts)-1 {
			fmt.Println("...waiting for 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("🚀 Thread posted successfully!")
	return nil
}

/*
func main() {
	client, err := NewClient()
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// --- Example Usage ---
	// Define the content of your tweet thread
	threadToPost := []string{
		"This is the first tweet in a thread, created using Go! 🧵",
		"Here's the second part. I'm using the g8rswimmer/go-twitter/v2 and dghubble/oauth1 libraries.",
		"And this is the final tweet. The Go code automatically replies to the previous one to build the thread. #golang",
	}

	// Post the entire thread
	if err := client.PostTweetThread(threadToPost); err != nil {
		log.Fatalf("Error posting thread: %v", err)
	}
}
*/
