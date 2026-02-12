package bitbucket

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

// mockRoundTripper implements http.RoundTripper for testing HTTP requests.
type mockRoundTripper struct {
	lastRequest  *http.Request
	lastBody     []byte
	responseCode int
	responseBody string
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req
	if req.Body != nil {
		body, _ := ioutil.ReadAll(req.Body)
		m.lastBody = body
	}
	resp := &http.Response{
		StatusCode: m.responseCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(m.responseBody)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestPostInlineComment_Success(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusCreated,
		responseBody: `{"id": 1}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	// Patch http.DefaultClient.Transport for this test
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	err := client.PostInlineComment("123", "foo.go", 42, "Test inline comment")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastRequest == nil {
		t.Fatal("expected request to be made")
	}
	if mock.lastRequest.Method != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastRequest.Method)
	}
	if !bytes.Contains(mock.lastBody, []byte(`"foo.go"`)) {
		t.Errorf("expected file path in body, got %s", string(mock.lastBody))
	}
	if !bytes.Contains(mock.lastBody, []byte(`"Test inline comment"`)) {
		t.Errorf("expected comment text in body, got %s", string(mock.lastBody))
	}
}

func TestPostInlineComment_Failure(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusBadRequest,
		responseBody: `{"error": "bad request"}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	err := client.PostInlineComment("123", "foo.go", 42, "Test inline comment")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.lastRequest == nil {
		t.Fatal("expected request to be made")
	}
}

func TestPostSummaryComment_Success(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusCreated,
		responseBody: `{"id": 2}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	err := client.PostSummaryComment("123", "This is a summary comment")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.lastRequest == nil {
		t.Fatal("expected request to be made")
	}
	if mock.lastRequest.Method != "POST" {
		t.Errorf("expected POST method, got %s", mock.lastRequest.Method)
	}
	if !bytes.Contains(mock.lastBody, []byte(`"This is a summary comment"`)) {
		t.Errorf("expected summary text in body, got %s", string(mock.lastBody))
	}
}

func TestPostSummaryComment_Failure(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusBadRequest,
		responseBody: `{"error": "bad request"}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	err := client.PostSummaryComment("123", "This is a summary comment")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.lastRequest == nil {
		t.Fatal("expected request to be made")
	}
}

func TestCreatePullRequest_Success(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusCreated,
		responseBody: `{
			"id": 42,
			"title": "Test PR",
			"state": "OPEN",
			"links": {
				"html": {
					"href": "https://bitbucket.org/ws/repo/pull-requests/42"
				}
			}
		}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	req := CreatePullRequestRequest{
		Title:             "Test PR",
		Description:       "Test Description",
		SourceBranch:      "feature-branch",
		DestinationBranch: "main",
		CloseSourceBranch: true,
	}

	resp, err := client.CreatePullRequest(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != 42 {
		t.Errorf("expected ID 42, got %d", resp.ID)
	}
	if resp.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %s", resp.Title)
	}
	if !bytes.Contains(mock.lastBody, []byte(`"feature-branch"`)) {
		t.Errorf("expected source branch in body, got %s", string(mock.lastBody))
	}
}

func TestCreatePullRequest_BranchNotFound(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusNotFound,
		responseBody: `{"error": {"message": "Branch not found"}}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	req := CreatePullRequestRequest{
		Title:             "Test PR",
		Description:       "Test Description",
		SourceBranch:      "nonexistent-branch",
		DestinationBranch: "main",
		CloseSourceBranch: true,
	}

	_, err := client.CreatePullRequest(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreatePullRequest_InsufficientPermissions(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusForbidden,
		responseBody: `{"error": {"message": "Forbidden"}}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	req := CreatePullRequestRequest{
		Title:             "Test PR",
		Description:       "Test Description",
		SourceBranch:      "feature-branch",
		DestinationBranch: "main",
		CloseSourceBranch: true,
	}

	_, err := client.CreatePullRequest(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFileContent_Success(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusOK,
		responseBody: "package main\n\nfunc main() {}\n",
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	content, err := client.GetFileContent(ctx, "main", "main.go")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if content != "package main\n\nfunc main() {}\n" {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGetFileContent_FileNotFound(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusNotFound,
		responseBody: `{"error": {"message": "File not found"}}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	_, err := client.GetFileContent(ctx, "main", "nonexistent.go")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBranchExists_True(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusOK,
		responseBody: `{"name": "feature-branch"}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	exists, err := client.BranchExists(ctx, "feature-branch")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !exists {
		t.Error("expected branch to exist")
	}
}

func TestBranchExists_False(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusNotFound,
		responseBody: `{"error": {"message": "Branch not found"}}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	exists, err := client.BranchExists(ctx, "nonexistent-branch")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if exists {
		t.Error("expected branch not to exist")
	}
}

func TestGetPullRequestByBranch_Found(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusOK,
		responseBody: `{
			"values": [{
				"id": 42,
				"title": "Test PR",
				"state": "OPEN",
				"links": {
					"html": {
						"href": "https://bitbucket.org/ws/repo/pull-requests/42"
					}
				}
			}]
		}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	pr, err := client.GetPullRequestByBranch(ctx, "feature-branch")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pr == nil {
		t.Fatal("expected PR to be found")
	}

	if pr.ID != 42 {
		t.Errorf("expected ID 42, got %d", pr.ID)
	}
}

func TestGetPullRequestByBranch_NotFound(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusOK,
		responseBody: `{"values": []}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	pr, err := client.GetPullRequestByBranch(ctx, "nonexistent-branch")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pr != nil {
		t.Error("expected nil PR")
	}
}

func TestGetPullRequest_Success(t *testing.T) {
	mock := &mockRoundTripper{
		responseCode: http.StatusOK,
		responseBody: `{
			"id": 42,
			"title": "Test PR",
			"description": "Test Description",
			"state": "OPEN",
			"source": {
				"branch": {
					"name": "feature-branch"
				}
			},
			"destination": {
				"branch": {
					"name": "main"
				}
			},
			"author": {
				"display_name": "John Doe"
			},
			"links": {
				"html": {
					"href": "https://bitbucket.org/ws/repo/pull-requests/42"
				}
			}
		}`,
	}
	client := &Client{
		Email:     "user@example.com",
		APIToken:  "token",
		Workspace: "ws",
		RepoSlug:  "repo",
		BaseURL:   "https://api.bitbucket.org/2.0",
	}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = mock
	defer func() { http.DefaultClient.Transport = origTransport }()

	ctx := context.Background()
	pr, err := client.GetPullRequest(ctx, "42")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pr.ID != 42 {
		t.Errorf("expected ID 42, got %d", pr.ID)
	}
	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %s", pr.Title)
	}
	if pr.SourceBranch != "feature-branch" {
		t.Errorf("expected source branch 'feature-branch', got %s", pr.SourceBranch)
	}
	if pr.DestBranch != "main" {
		t.Errorf("expected dest branch 'main', got %s", pr.DestBranch)
	}
	if pr.Author != "John Doe" {
		t.Errorf("expected author 'John Doe', got %s", pr.Author)
	}
}
