package claudecode

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedModel string
	}{
		{name: "default model when empty", model: "", expectedModel: "sonnet"},
		{name: "default model when whitespace", model: "   ", expectedModel: "sonnet"},
		{name: "custom alias", model: "opus", expectedModel: "opus"},
		{name: "explicit model id", model: "claude-sonnet-4-6", expectedModel: "claude-sonnet-4-6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.model)
			if client.Model != tt.expectedModel {
				t.Errorf("NewClient(%q).Model = %q, want %q", tt.model, client.Model, tt.expectedModel)
			}
			if client.Timeout == 0 {
				t.Error("NewClient should set a default timeout")
			}
			if client.lookPath == nil {
				t.Error("NewClient should set a default lookPath")
			}
			if client.run == nil {
				t.Error("NewClient should set a default run")
			}
		})
	}
}

func TestParseAuthStatus(t *testing.T) {
	tests := []struct {
		name      string
		payload   string
		wantErr   bool
		errSubstr string
	}{
		{name: "logged in", payload: `{"loggedIn": true, "authMethod": "claude.ai"}`, wantErr: false},
		{name: "logged out", payload: `{"loggedIn": false}`, wantErr: true, errSubstr: "not authenticated"},
		{name: "missing field defaults to false", payload: `{"authMethod": "claude.ai"}`, wantErr: true, errSubstr: "not authenticated"},
		{name: "malformed JSON", payload: `{not json`, wantErr: true, errSubstr: "could not parse"},
		{name: "empty payload", payload: ``, wantErr: true, errSubstr: "could not parse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseAuthStatus([]byte(tt.payload))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAuthStatus(%q) = nil, want error containing %q", tt.payload, tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// fakeRun returns a runFunc that dispatches based on the args[0] subcommand
// of the claude invocation: "auth" for the auth-status check, anything else
// (in practice "-p") for the review send.
func fakeRun(authStdout, authStderr []byte, authErr error, sendStdout, sendStderr []byte, sendErr error) runFunc {
	return func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			return authStdout, authStderr, authErr
		}
		return sendStdout, sendStderr, sendErr
	}
}

// newTestClient builds a Client with the CLI checks already stubbed to "OK",
// so individual tests can override only the call they care about.
func newTestClient(run runFunc) *Client {
	c := NewClient("sonnet")
	c.lookPath = func(string) (string, error) { return "/usr/bin/claude", nil }
	c.run = run
	return c
}

func TestCheckCLIAvailable_NotFound(t *testing.T) {
	c := NewClient("")
	c.lookPath = func(string) (string, error) { return "", errors.New("not found") }
	err := c.checkCLIAvailable()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestCheckCLIAvailable_AuthError(t *testing.T) {
	c := newTestClient(fakeRun(nil, []byte("you are logged out"), errors.New("exit 1"), nil, nil, nil))
	err := c.checkCLIAvailable()
	if err == nil || !strings.Contains(err.Error(), "claude auth status failed") {
		t.Fatalf("expected auth status failure, got: %v", err)
	}
}

func TestCheckCLIAvailable_LoggedOut(t *testing.T) {
	c := newTestClient(fakeRun([]byte(`{"loggedIn": false}`), nil, nil, nil, nil, nil))
	err := c.checkCLIAvailable()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected logged-out error, got: %v", err)
	}
}

func TestCheckCLIAvailable_AuthTimeout(t *testing.T) {
	c := NewClient("")
	c.lookPath = func(string) (string, error) { return "/usr/bin/claude", nil }
	c.run = func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	// Force the auth probe to bound to 30ms via the overall Timeout.
	c.Timeout = 30 * time.Millisecond

	start := time.Now()
	err := c.checkAuth()
	elapsed := time.Since(start)

	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected auth timeout error, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("auth check did not respect timeout, elapsed=%v", elapsed)
	}
}

func TestCheckCLIAvailable_OK(t *testing.T) {
	c := newTestClient(fakeRun([]byte(`{"loggedIn": true}`), nil, nil, nil, nil, nil))
	if err := c.checkCLIAvailable(); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestEnsureCLIAvailableCachesResult(t *testing.T) {
	var calls int
	authOK := []byte(`{"loggedIn": true}`)
	c := newTestClient(func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			calls++
			return authOK, nil, nil
		}
		return nil, nil, nil
	})
	for i := range 5 {
		if err := c.ensureCLIAvailable(); err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}
	if calls != 1 {
		t.Errorf("expected auth status to be invoked exactly once, got %d", calls)
	}
}

func TestSendReviewPrompt_HappyPath(t *testing.T) {
	var sentStdin []byte
	var sentArgs []string
	c := newTestClient(func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			return []byte(`{"loggedIn": true}`), nil, nil
		}
		sentStdin = append([]byte(nil), stdin...)
		sentArgs = append([]string(nil), args...)
		return []byte("review body here\n"), nil, nil
	})

	out, err := c.SendReviewPrompt("please review this diff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "review body here" {
		t.Errorf("output = %q, want %q (trimmed)", out, "review body here")
	}
	if string(sentStdin) != "please review this diff" {
		t.Errorf("stdin sent to claude = %q, want %q", string(sentStdin), "please review this diff")
	}
	wantArgs := []string{"-p", "--model", "sonnet", "--tools", ""}
	if !slices.Equal(sentArgs, wantArgs) {
		t.Errorf("args = %v, want %v", sentArgs, wantArgs)
	}
}

func TestSendReviewPrompt_NonZeroExit(t *testing.T) {
	c := newTestClient(fakeRun(
		[]byte(`{"loggedIn": true}`), nil, nil,
		nil, []byte("rate limit exceeded"), errors.New("exit status 1"),
	))
	_, err := c.SendReviewPrompt("hi")
	if err == nil {
		t.Fatal("expected error from non-zero exit, got nil")
	}
	if !strings.Contains(err.Error(), "claude CLI failed") {
		t.Errorf("error = %q, want 'claude CLI failed' prefix", err.Error())
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("error should include stderr 'rate limit exceeded', got %q", err.Error())
	}
}

func TestSendReviewPrompt_EmptyStdout(t *testing.T) {
	c := newTestClient(fakeRun(
		[]byte(`{"loggedIn": true}`), nil, nil,
		[]byte("   \n"), nil, nil,
	))
	_, err := c.SendReviewPrompt("hi")
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected 'empty response' error, got: %v", err)
	}
}

func TestSendReviewPrompt_Timeout(t *testing.T) {
	c := newTestClient(func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			return []byte(`{"loggedIn": true}`), nil, nil
		}
		// Simulate a long-running subprocess that respects context cancellation.
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return []byte("late"), nil, nil
		}
	})
	c.Timeout = 20 * time.Millisecond

	_, err := c.SendReviewPrompt("hi")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestSendReviewPrompt_TimeoutIncludesStderr(t *testing.T) {
	c := newTestClient(func(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
		if len(args) > 0 && args[0] == "auth" {
			return []byte(`{"loggedIn": true}`), nil, nil
		}
		select {
		case <-ctx.Done():
			return nil, []byte("partial output before kill\n"), ctx.Err()
		case <-time.After(5 * time.Second):
			return []byte("late"), nil, nil
		}
	})
	c.Timeout = 20 * time.Millisecond

	_, err := c.SendReviewPrompt("hi")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want 'timed out'", err.Error())
	}
	if !strings.Contains(err.Error(), "partial output before kill") {
		t.Errorf("timeout error should surface stderr; got %q", err.Error())
	}
}

func TestSendReviewPrompt_AuthFailureSurfaces(t *testing.T) {
	c := newTestClient(fakeRun(
		[]byte(`{"loggedIn": false}`), nil, nil,
		nil, nil, fmt.Errorf("should not reach here"),
	))
	_, err := c.SendReviewPrompt("hi")
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected not-authenticated error, got: %v", err)
	}
}

func TestSetVerbose(t *testing.T) {
	SetVerbose(true)
	if !verbose() {
		t.Error("SetVerbose(true) should make verbose() return true")
	}
	SetVerbose(false)
	if verbose() {
		t.Error("SetVerbose(false) should make verbose() return false")
	}
}
