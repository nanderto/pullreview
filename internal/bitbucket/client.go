package bitbucket

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Client provides methods for interacting with the Bitbucket Cloud API.
type Client struct {
	Email     string
	APIToken  string
	Workspace string
	BaseURL   string
}

// NewClient creates a new Bitbucket API client.
func NewClient(email, apiToken, workspace, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.bitbucket.org/2.0"
	}
	return &Client{
		Email:     email,
		APIToken:  apiToken,
		Workspace: workspace,
		BaseURL:   baseURL,
	}
}

// Authenticate checks if the Bitbucket credentials are valid by calling the /user endpoint.
// Returns nil if authentication is successful, or an error with details otherwise.
func (c *Client) Authenticate() error {
	if c.Email == "" {
		return errors.New("missing Bitbucket account email")
	}
	if c.APIToken == "" {
		return errors.New("missing Bitbucket API token")
	}

	req, err := http.NewRequest("GET", c.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create authentication request: %w", err)
	}

	// ✅ Use email as username and API token as password
	req.SetBasicAuth(c.Email, c.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact Bitbucket API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed: invalid Bitbucket credentials. Response: %s", bodyStr)
	default:
		return fmt.Errorf("authentication failed: Bitbucket API returned status %d. Response: %s",
			resp.StatusCode, bodyStr)
	}
}

// Placeholder for future Bitbucket API methods.
