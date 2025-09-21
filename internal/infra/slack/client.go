package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	webhookURL string
	timeout    time.Duration
	httpClient *http.Client
}

type SlackMessage struct {
	Text string `json:"text"`
}

func NewClient(webhookURL string, timeout time.Duration) *Client {
	return &Client{
		webhookURL: webhookURL,
		timeout:    timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) SendMessage(message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	payload := SlackMessage{
		Text: message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to send message: received status code %d", resp.StatusCode)
	}

	return nil
}
