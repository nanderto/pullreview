// Package claudecode provides a client that runs reviews through the locally
// installed Claude Code CLI (`claude`) instead of a hosted LLM API.
package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// verboseMode is set by SetVerbose. atomic.Bool makes concurrent reads/writes safe
// even though pullreview is normally single-shot — `go test -race` will flag any
// concurrent access otherwise.
var verboseMode atomic.Bool

func verbose() bool { return verboseMode.Load() }

// runFunc executes an external command with the given stdin and returns its
// stdout, stderr, and run error. Injected for testability.
type runFunc func(ctx context.Context, stdin []byte, name string, args ...string) (stdout, stderr []byte, err error)

// Client invokes the Claude Code CLI to generate review responses.
type Client struct {
	Model   string        // Model name or alias (e.g., "sonnet", "opus", "claude-sonnet-4-6")
	Timeout time.Duration // Timeout for a single CLI invocation

	// lookPath resolves the claude binary on PATH. Defaults to exec.LookPath.
	lookPath func(string) (string, error)
	// run executes a subprocess. Defaults to a real os/exec invocation.
	run runFunc

	checkOnce sync.Once
	checkErr  error
}

// NewClient creates a new Claude Code CLI client. An empty model defaults to "sonnet".
func NewClient(model string) *Client {
	if strings.TrimSpace(model) == "" {
		model = "sonnet"
	}
	return &Client{
		Model:    model,
		Timeout:  5 * time.Minute,
		lookPath: exec.LookPath,
		run:      defaultRun,
	}
}

// defaultRun is the production implementation of runFunc.
func defaultRun(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// defaultAuthProbeTimeout caps the `claude auth status` subprocess so a stuck
// credential refresh (e.g. network I/O) can't hang the whole review.
const defaultAuthProbeTimeout = 10 * time.Second

// checkCLIAvailable verifies that the Claude Code CLI is installed and the user is logged in.
func (c *Client) checkCLIAvailable() error {
	if _, err := c.lookPath("claude"); err != nil {
		return errors.New("claude CLI not found, install from https://docs.anthropic.com/claude-code and ensure 'claude' is in your PATH")
	}
	return c.checkAuth()
}

// authProbeTimeout returns the smaller of the client's overall Timeout and the
// default auth-probe ceiling, so the probe is always bounded.
func (c *Client) authProbeTimeout() time.Duration {
	if c.Timeout > 0 && c.Timeout < defaultAuthProbeTimeout {
		return c.Timeout
	}
	return defaultAuthProbeTimeout
}

// checkAuth runs `claude auth status --json` and confirms the user is logged in.
// The Claude Code CLI ships a first-party auth-status command, so we avoid the
// "send a hello prompt" probe used by the Copilot integration.
func (c *Client) checkAuth() error {
	timeout := c.authProbeTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout, stderr, err := c.run(ctx, nil, "claude", "auth", "status", "--json")
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("claude auth status timed out after %v", timeout)
		}
		return fmt.Errorf("claude auth status failed: %s: %w", strings.TrimSpace(string(stderr)), err)
	}
	return parseAuthStatus(stdout)
}

// parseAuthStatus parses the JSON payload emitted by `claude auth status --json`
// and returns nil iff the user is logged in. Extracted for testability.
func parseAuthStatus(payload []byte) error {
	// Only parse loggedIn; ignore the rest so future schema additions don't break us.
	var status struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return fmt.Errorf("could not parse claude auth status output: %w", err)
	}
	if !status.LoggedIn {
		return errors.New("claude CLI is not authenticated, run 'claude auth login' to sign in")
	}
	return nil
}

// CheckCLIAvailable verifies that the Claude Code CLI is installed and the user is logged in,
// using the default exec implementation. Provided for callers that want to probe availability
// without constructing a Client.
func CheckCLIAvailable() error {
	return NewClient("").checkCLIAvailable()
}

// ensureCLIAvailable runs the availability check once per Client and caches the result,
// so repeated SendReviewPrompt calls don't spawn an auth-status subprocess each time.
func (c *Client) ensureCLIAvailable() error {
	c.checkOnce.Do(func() {
		c.checkErr = c.checkCLIAvailable()
	})
	return c.checkErr
}

// SendReviewPrompt sends the review prompt to Claude Code via stdin and returns stdout.
func (c *Client) SendReviewPrompt(prompt string) (string, error) {
	if err := c.ensureCLIAvailable(); err != nil {
		return "", err
	}

	if verbose() {
		fmt.Fprintf(os.Stderr, "[claudecode] Model: %s\n", c.Model)
		fmt.Fprintf(os.Stderr, "[claudecode] Timeout: %v\n", c.Timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	if verbose() {
		fmt.Fprintln(os.Stderr, "[claudecode] Invoking claude -p ...")
	}

	// `--tools ""` disables all tools so the run is a pure text completion — faster,
	// predictable, and won't touch the working directory.
	stdout, stderr, err := c.run(ctx, []byte(prompt), "claude", "-p", "--model", c.Model, "--tools", "")
	if err != nil {
		stderrTrimmed := strings.TrimSpace(string(stderr))
		if ctx.Err() == context.DeadlineExceeded {
			if stderrTrimmed != "" {
				return "", fmt.Errorf("claude CLI timed out after %v: %s", c.Timeout, stderrTrimmed)
			}
			return "", fmt.Errorf("claude CLI timed out after %v", c.Timeout)
		}
		return "", fmt.Errorf("claude CLI failed: %s: %w", stderrTrimmed, err)
	}

	out := strings.TrimSpace(string(stdout))
	if out == "" {
		return "", errors.New("empty response received from claude CLI")
	}

	if verbose() {
		fmt.Fprintln(os.Stderr, "[claudecode] Response received successfully")
	}
	return out, nil
}

// SetVerbose enables or disables verbose mode for Claude Code debug output.
func SetVerbose(v bool) {
	verboseMode.Store(v)
}
