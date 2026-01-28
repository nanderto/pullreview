package bitbucket

import (
	"encoding/json"
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
	RepoSlug  string
	BaseURL   string
}

// NewClient creates a new Bitbucket API client.
func NewClient(email, apiToken, workspace, repoSlug, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.bitbucket.org/2.0"
	}
	return &Client{
		Email:     email,
		APIToken:  apiToken,
		Workspace: workspace,
		RepoSlug:  repoSlug,
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

// GetPRIDByBranch fetches the PR ID associated with the given branch in the workspace/repo.
// Returns the PR ID as a string, or an error if not found or on failure.
func (c *Client) GetPRIDByBranch(branch string) (string, error) {
	if branch == "" {
		return "", errors.New("branch name is required")
	}
	if c.RepoSlug == "" {
		return "", errors.New("repo slug is required")
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests?q=source.branch.name=\"%s\"&state=OPEN", c.BaseURL, c.Workspace, c.RepoSlug, branch)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create PR lookup request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact Bitbucket API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch PRs: status %d, response: %s", resp.StatusCode, string(body))
	}
	type prList struct {
		Values []struct {
			ID     int    `json:"id"`
			Title  string `json:"title"`
			State  string `json:"state"`
			Source struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"name"`
			} `json:"source"`
		} `json:"values"`
	}
	var prs prList
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&prs); err != nil {
		return "", fmt.Errorf("failed to decode PR list: %w", err)
	}
	if len(prs.Values) == 0 {
		return "", fmt.Errorf("no open PR found for branch %q", branch)
	}
	return fmt.Sprintf("%d", prs.Values[0].ID), nil
}

// GetPRMetadata fetches metadata for a given PR ID.
// Returns the raw JSON response as bytes, or an error.
func (c *Client) GetPRMetadata(prID string) ([]byte, error) {
	if prID == "" {
		return nil, errors.New("PR ID is required")
	}
	if c.RepoSlug == "" {
		return nil, errors.New("repo slug is required")
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%s", c.BaseURL, c.Workspace, c.RepoSlug, prID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR metadata request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Bitbucket API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch PR metadata: status %d, response: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// GetPRDiff fetches the unified diff for a given PR ID.
// Returns the diff as a string, or an error.
func (c *Client) GetPRDiff(prID string) (string, error) {
	if prID == "" {
		return "", errors.New("PR ID is required")
	}
	if c.RepoSlug == "" {
		return "", errors.New("repo slug is required")
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%s/diff", c.BaseURL, c.Workspace, c.RepoSlug, prID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create PR diff request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact Bitbucket API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch PR diff: status %d, response: %s", resp.StatusCode, string(body))
	}
	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PR diff: %w", err)
	}
	return string(diffBytes), nil
}
