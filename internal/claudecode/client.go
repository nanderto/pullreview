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
	"time"
)

var verboseMode bool

// Client invokes the Claude Code CLI to generate review responses.
type Client struct {
	Model   string        // Model name or alias (e.g., "sonnet", "opus", "claude-sonnet-4-6")
	Timeout time.Duration // Timeout for a single CLI invocation
}

// NewClient creates a new Claude Code CLI client. An empty model defaults to "sonnet".
func NewClient(model string) *Client {
	if strings.TrimSpace(model) == "" {
		model = "sonnet"
	}
	return &Client{
		Model:   model,
		Timeout: 5 * time.Minute,
	}
}

// CheckCLIAvailable verifies that the Claude Code CLI is installed and the user is logged in.
func CheckCLIAvailable() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.New("Claude Code CLI not found. Install from https://docs.anthropic.com/claude-code and ensure 'claude' is in your PATH")
	}
	return checkAuth()
}

// checkAuth runs `claude auth status --json` and confirms the user is logged in.
// The Claude Code CLI ships a first-party auth-status command, so we avoid the
// "send a hello prompt" probe used by the Copilot integration.
func checkAuth() error {
	cmd := exec.Command("claude", "auth", "status", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude auth status failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	// Only parse loggedIn; ignore the rest so future schema additions don't break us.
	var status struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		return fmt.Errorf("could not parse claude auth status output: %w", err)
	}
	if !status.LoggedIn {
		return errors.New("Claude Code CLI is not authenticated. Run 'claude auth login' to sign in")
	}
	return nil
}

// SendReviewPrompt sends the review prompt to Claude Code via stdin and returns stdout.
func (c *Client) SendReviewPrompt(prompt string) (string, error) {
	if err := CheckCLIAvailable(); err != nil {
		return "", err
	}

	if verboseMode {
		fmt.Fprintf(os.Stderr, "[claudecode] Model: %s\n", c.Model)
		fmt.Fprintf(os.Stderr, "[claudecode] Timeout: %v\n", c.Timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// `--tools ""` disables all tools so the run is a pure text completion — faster,
	// predictable, and won't touch the working directory.
	cmd := exec.CommandContext(ctx, "claude", "-p", "--model", c.Model, "--tools", "")
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if verboseMode {
		fmt.Fprintln(os.Stderr, "[claudecode] Invoking claude -p ...")
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("claude CLI timed out after %v", c.Timeout)
		}
		return "", fmt.Errorf("claude CLI failed: %s: %w", strings.TrimSpace(stderr.String()), err)
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", errors.New("empty response received from Claude Code CLI")
	}

	if verboseMode {
		fmt.Fprintln(os.Stderr, "[claudecode] Response received successfully")
	}
	return out, nil
}

// SetVerbose enables or disables verbose mode for Claude Code debug output.
func SetVerbose(v bool) {
	verboseMode = v
}
