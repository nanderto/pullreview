package bitbucket

import (
	"bytes"
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
