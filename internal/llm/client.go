package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client provides methods for interacting with a Large Language Model (LLM) API.

type Client struct {
	Provider string
	APIKey   string
	Endpoint string
	Model    string // LLM model name (e.g., arcee-ai/trinity-large-preview:free)
}

// NewClient creates a new LLM API client.

func NewClient(provider, apiKey, endpoint string) *Client {

	return &Client{

		Provider: provider,

		APIKey: apiKey,

		Endpoint: endpoint,
	}

}

// ReviewRequest represents the input for an LLM review.
type ReviewRequest struct {
	Prompt string
}

// ReviewResponse represents the output from an LLM review.
type ReviewResponse struct {
	Content string
}

// SendReviewPrompt sends the review prompt to the configured LLM provider and returns the response.
func (c *Client) SendReviewPrompt(prompt string) (string, error) {
	switch strings.ToLower(c.Provider) {
	case "openai", "openrouter":
		return c.sendOpenAI(prompt)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.Provider)
	}
}

// sendOpenAI sends the prompt to OpenAI's Chat API and returns the response.
func (c *Client) sendOpenAI(prompt string) (string, error) {
	if c.APIKey == "" {
		return "", errors.New("missing OpenAI API key")
	}
	if c.Endpoint == "" {
		return "", errors.New("missing OpenAI API endpoint")
	}

	model := c.Model

	if model == "" {

		model = "gpt-3.5-turbo"

	}

	// Prepare request body for OpenAI/OpenRouter Chat API
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
		"max_tokens":  2048,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenAI response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	// Parse OpenAI response
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
	}
	if len(openAIResp.Choices) == 0 {
		return "", errors.New("no choices returned from OpenAI API")
	}
	return openAIResp.Choices[0].Message.Content, nil
}
