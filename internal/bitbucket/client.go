package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// PRComment represents a comment to be posted to a PR.
type PRComment struct {
	FilePath string // Relative file path for inline comments
	Line     int    // Line number for inline comments (new file)
	Text     string // Markdown comment text
}

// PostInlineComment posts an inline comment to a specific line in a PR.
func (c *Client) PostInlineComment(prID, filePath string, line int, text string) error {
	if prID == "" || filePath == "" || line <= 0 || text == "" {
		return errors.New("missing required fields for inline comment")
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%s/comments", c.BaseURL, c.Workspace, c.RepoSlug, prID)
	body := map[string]interface{}{
		"content": map[string]string{
			"raw": text,
		},
		"inline": map[string]interface{}{
			"path": filePath,
			"to":   line,
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal inline comment: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create inline comment request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post inline comment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to post inline comment: status %d, response: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// PostSummaryComment posts a summary (top-level) comment to a PR.
func (c *Client) PostSummaryComment(prID, text string) error {
	if prID == "" || text == "" {
		return errors.New("missing required fields for summary comment")
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%s/comments", c.BaseURL, c.Workspace, c.RepoSlug, prID)
	body := map[string]interface{}{
		"content": map[string]string{
			"raw": text,
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal summary comment: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create summary comment request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post summary comment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to post summary comment: status %d, response: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

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

// PullRequest represents a Bitbucket pull request.
type PullRequest struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	State        string `json:"state"`
	SourceBranch string
	DestBranch   string
	Author       string
	Links        struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// CreatePullRequestRequest represents a PR creation request.
type CreatePullRequestRequest struct {
	Title             string
	Description       string
	SourceBranch      string
	DestinationBranch string
	CloseSourceBranch bool
	Reviewers         []string // Optional: usernames to add as reviewers
}

// CreatePullRequestResponse represents the API response for PR creation.
type CreatePullRequestResponse struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// CreatePullRequest creates a new pull request in Bitbucket.
// Creates a stacked PR targeting the specified destination branch.
func (c *Client) CreatePullRequest(ctx context.Context, req CreatePullRequestRequest) (*CreatePullRequestResponse, error) {
	if req.Title == "" {
		return nil, errors.New("PR title is required")
	}
	if req.SourceBranch == "" {
		return nil, errors.New("source branch is required")
	}
	if req.DestinationBranch == "" {
		return nil, errors.New("destination branch is required")
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests", c.BaseURL, c.Workspace, c.RepoSlug)

	// Build request body
	body := map[string]interface{}{
		"title":       req.Title,
		"description": req.Description,
		"source": map[string]interface{}{
			"branch": map[string]string{
				"name": req.SourceBranch,
			},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{
				"name": req.DestinationBranch,
			},
		},
		"close_source_branch": req.CloseSourceBranch,
	}

	// Add reviewers if provided
	if len(req.Reviewers) > 0 {
		reviewers := make([]map[string]string, 0, len(req.Reviewers))
		for _, username := range req.Reviewers {
			reviewers = append(reviewers, map[string]string{"username": username})
		}
		body["reviewers"] = reviewers
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PR request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create PR request: %w", err)
	}

	httpReq.SetBasicAuth(c.Email, c.APIToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create PR: status %d, response: %s", resp.StatusCode, string(respBody))
	}

	var prResp CreatePullRequestResponse
	if err := json.Unmarshal(respBody, &prResp); err != nil {
		return nil, fmt.Errorf("failed to decode PR response: %w", err)
	}

	return &prResp, nil
}

// GetFileContent fetches the content of a file from a specific branch.
// Used to read current file contents after fixes are applied.
func (c *Client) GetFileContent(ctx context.Context, branch string, filePath string) (string, error) {
	if branch == "" {
		return "", errors.New("branch name is required")
	}
	if filePath == "" {
		return "", errors.New("file path is required")
	}

	// URL encode the file path
	encodedPath := url.PathEscape(filePath)
	url := fmt.Sprintf("%s/repositories/%s/%s/src/%s/%s", c.BaseURL, c.Workspace, c.RepoSlug, branch, encodedPath)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create file content request: %w", err)
	}

	httpReq.SetBasicAuth(c.Email, c.APIToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch file content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s on branch %s", filePath, branch)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch file content: status %d, response: %s", resp.StatusCode, string(body))
	}

	contentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	return string(contentBytes), nil
}

// BranchExists checks if a branch exists in the remote repository.
func (c *Client) BranchExists(ctx context.Context, branchName string) (bool, error) {
	if branchName == "" {
		return false, errors.New("branch name is required")
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/refs/branches/%s", c.BaseURL, c.Workspace, c.RepoSlug, branchName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create branch check request: %w", err)
	}

	httpReq.SetBasicAuth(c.Email, c.APIToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("unexpected response checking branch: status %d, response: %s", resp.StatusCode, string(body))
}

// GetPullRequestByBranch finds a PR by its source branch name.
// Returns nil if no PR found.
func (c *Client) GetPullRequestByBranch(ctx context.Context, sourceBranch string) (*PullRequest, error) {
	if sourceBranch == "" {
		return nil, errors.New("source branch is required")
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests?q=source.branch.name=\"%s\"",
		c.BaseURL, c.Workspace, c.RepoSlug, sourceBranch)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR search request: %w", err)
	}

	httpReq.SetBasicAuth(c.Email, c.APIToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search for PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search for PR: status %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Values []PullRequest `json:"values"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode PR search response: %w", err)
	}

	if len(result.Values) == 0 {
		return nil, nil
	}

	// Return first matching PR
	return &result.Values[0], nil
}

// GetPullRequest fetches full PR details by PR ID.
func (c *Client) GetPullRequest(ctx context.Context, prID string) (*PullRequest, error) {
	if prID == "" {
		return nil, errors.New("PR ID is required")
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%s", c.BaseURL, c.Workspace, c.RepoSlug, prID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR request: %w", err)
	}

	httpReq.SetBasicAuth(c.Email, c.APIToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch PR: status %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode PR response: %w", err)
	}

	pr := &PullRequest{}

	// Extract fields safely
	if id, ok := result["id"].(float64); ok {
		pr.ID = int(id)
	}
	if title, ok := result["title"].(string); ok {
		pr.Title = title
	}
	if desc, ok := result["description"].(string); ok {
		pr.Description = desc
	}
	if state, ok := result["state"].(string); ok {
		pr.State = state
	}

	// Extract source branch
	if source, ok := result["source"].(map[string]interface{}); ok {
		if branch, ok := source["branch"].(map[string]interface{}); ok {
			if name, ok := branch["name"].(string); ok {
				pr.SourceBranch = name
			}
		}
	}

	// Extract destination branch
	if dest, ok := result["destination"].(map[string]interface{}); ok {
		if branch, ok := dest["branch"].(map[string]interface{}); ok {
			if name, ok := branch["name"].(string); ok {
				pr.DestBranch = name
			}
		}
	}

	// Extract author
	if author, ok := result["author"].(map[string]interface{}); ok {
		if displayName, ok := author["display_name"].(string); ok {
			pr.Author = displayName
		}
	}

	// Extract links
	if links, ok := result["links"].(map[string]interface{}); ok {
		if html, ok := links["html"].(map[string]interface{}); ok {
			if href, ok := html["href"].(string); ok {
				pr.Links.HTML.Href = href
			}
		}
	}

	return pr, nil
}
