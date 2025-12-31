package ntfy

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

const defaultServerURL = "https://ntfy.sh"

// Client sends notifications via ntfy
type Client struct {
	ServerURL string
	Topic     string
	Token     string
}

// NewClient creates a new ntfy client
func NewClient(serverURL, topic, token string) *Client {
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	return &Client{
		ServerURL: serverURL,
		Topic:     topic,
		Token:     token,
	}
}

// IsConfigured returns true if the topic is set
func (c *Client) IsConfigured() bool {
	return c.Topic != "" && c.ServerURL != ""
}

// Send sends a notification with the given title and message
func (c *Client) Send(title, message string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("ntfy credentials not configured")
	}

	url := strings.TrimRight(c.ServerURL, "/") + "/" + strings.TrimLeft(c.Topic, "/")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("failed to build notification request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	if title != "" {
		req.Header.Set("Title", title)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}

// Test sends a test notification to verify credentials
func (c *Client) Test() error {
	return c.Send("Shrinkray", "Test notification - ntfy is configured correctly!")
}
