package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"pullreview/internal/copilot"
	"strings"
)

var verboseMode bool

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
	// Always print provider and model to stdout before sending the prompt
	model := c.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	fmt.Fprintf(os.Stdout, "[llm] Using provider %q with model %q\n", c.Provider, model)

	switch strings.ToLower(c.Provider) {
	case "openai", "openrouter":
		return c.sendOpenAI(prompt)
	case "copilot":
		return c.sendCopilot(prompt)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.Provider)
	}
}

// sendCopilot sends the prompt to GitHub Copilot via the SDK and returns the response.
func (c *Client) sendCopilot(prompt string) (string, error) {
	// Set verbose mode on the copilot package to match our setting
	copilot.SetVerbose(verboseMode)

	// Create a Copilot client with the configured model
	copilotClient := copilot.NewClient(c.Model)

	if verboseMode {
		fmt.Fprintf(os.Stderr, "[llm] Provider: %s\n", c.Provider)
		fmt.Fprintf(os.Stderr, "[llm] Model: %s\n", c.Model)
	}

	return copilotClient.SendReviewPrompt(prompt)
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

	// Print LLM config before making the API call, but only if verbose is enabled
	if verboseMode {
		fmt.Fprintf(os.Stderr, "[llm] Provider: %s\n", c.Provider)
		fmt.Fprintf(os.Stderr, "[llm] API Key: %s\n", c.APIKey)
		fmt.Fprintf(os.Stderr, "[llm] Endpoint: %s\n", c.Endpoint)
		fmt.Fprintf(os.Stderr, "[llm] Model: %s\n", model)
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
		// Try to parse OpenRouter-style error details
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBody, &errorResponse)
		if verboseMode {
			fmt.Fprintf(os.Stderr, "==============================================================================================================================\n")
			fmt.Fprintf(os.Stderr, "[llm] Raw error response from LLM:\n%s\n", string(respBody))
			fmt.Fprintf(os.Stderr, "==============================================================================================================================\n")
			fmt.Fprintf(os.Stderr, "[llm] Error response from LLM (parsed):\n")
			fmt.Fprintf(os.Stderr, "[llm]   Message: %s\n", errorResponse.Error.Message)
			fmt.Fprintf(os.Stderr, "[llm]   Type: %s\n", errorResponse.Error.Type)
			fmt.Fprintf(os.Stderr, "[llm]   Code: %s\n", errorResponse.Error.Code)
		}
		providerName := "OpenRouter"
		if strings.ToLower(c.Provider) == "openai" {
			providerName = "OpenAI"
		}
		return "", fmt.Errorf("%s API error: %s (type: %s, code: %s)",
			providerName,
			errorResponse.Error.Message,
			errorResponse.Error.Type,
			errorResponse.Error.Code)
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
	if verboseMode {
		fmt.Fprintf(os.Stdout, "==============================================================================================================================\n")
		fmt.Fprintf(os.Stdout, "[llm] Raw success response from LLM:\n")
		fmt.Fprintf(os.Stdout, "==============================================================================================================================\n\n")
		fmt.Fprintf(os.Stdout, "%s\n", string(respBody))
		fmt.Fprintf(os.Stdout, "\n===============================================================================================================================\n")
		fmt.Fprintf(os.Stdout, "===============================================================================================================================\n")
	}
	if len(openAIResp.Choices) == 0 {
		return "", errors.New("no choices returned from OpenAI API")
	}
	return openAIResp.Choices[0].Message.Content, nil
}

// SetVerbose enables or disables verbose mode for LLM debug output.
func SetVerbose(v bool) {
	verboseMode = v
}
