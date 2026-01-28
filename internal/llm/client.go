package llm

// Client provides methods for interacting with a Large Language Model (LLM) API.
type Client struct {
	Provider string
	APIKey   string
	Endpoint string
}

// NewClient creates a new LLM API client.
func NewClient(provider, apiKey, endpoint string) *Client {
	return &Client{
		Provider: provider,
		APIKey:   apiKey,
		Endpoint: endpoint,
	}
}

// Placeholder for future LLM API methods.
