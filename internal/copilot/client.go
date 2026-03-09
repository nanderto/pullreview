package copilot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

var verboseMode bool

// Client provides methods for interacting with GitHub Copilot via the SDK.
type Client struct {
	Model   string        // Model name (e.g., "gpt-4.1", "gpt-5")
	Timeout time.Duration // Timeout for Copilot requests
}

// NewClient creates a new GitHub Copilot SDK client.
func NewClient(model string) *Client {
	if model == "" {
		model = "gpt-4.1"
	}
	return &Client{
		Model:   model,
		Timeout: 5 * time.Minute,
	}
}

// CheckCLIAvailable verifies that the Copilot CLI is installed and accessible.
func CheckCLIAvailable() error {
	_, err := exec.LookPath("copilot")
	if err != nil {
		return errors.New("Copilot CLI not found. Please install from https://github.com/github/copilot-cli and ensure it is in your PATH")
	}

	// Check if Copilot CLI is authenticated
	if err := checkAuth(); err != nil {
		return err
	}

	return nil
}

// checkAuth verifies that the Copilot CLI is authenticated by running a test prompt.
// It checks both the exit code and stdout output, since some Copilot CLI versions
// exit with code 0 even on auth failure (but always produce empty stdout).
func checkAuth() error {
	cmd := exec.Command("copilot", "-p", "hello")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// Non-zero exit code means auth failed; empty stdout with exit code 0
	// also indicates auth failure (some CLI versions exit 0 on auth errors).
	if err != nil || stdout.Len() == 0 {
		return fmt.Errorf("copilot auth check failed: %s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

// SendReviewPrompt sends the review prompt to GitHub Copilot and returns the response.
func (c *Client) SendReviewPrompt(prompt string) (string, error) {
	// Check if Copilot CLI is available and authenticated
	if err := CheckCLIAvailable(); err != nil {
		return "", err
	}

	if verboseMode {
		fmt.Fprintf(os.Stderr, "[copilot] Model: %s\n", c.Model)
		fmt.Fprintf(os.Stderr, "[copilot] Timeout: %v\n", c.Timeout)
	}

	// Create the Copilot SDK client
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel: getLogLevel(),
	})

	// Create a context with the configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// Start the Copilot CLI server
	if verboseMode {
		fmt.Fprintln(os.Stderr, "[copilot] Starting Copilot CLI server...")
	}
	if err := client.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start Copilot CLI: %w", err)
	}
	defer client.Stop()

	// Create a session with the specified model
	sessionConfig := &copilot.SessionConfig{
		Model:     c.Model,
		Streaming: false, // We want the full response, not streaming
	}

	if verboseMode {
		fmt.Fprintln(os.Stderr, "[copilot] Creating session...")
	}
	session, err := client.CreateSession(ctx, sessionConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create Copilot session: %w", err)
	}

	// Send the prompt and wait for response
	if verboseMode {
		fmt.Fprintln(os.Stderr, "[copilot] Sending prompt to Copilot...")
	}
	// session.SendAndWait will wait indefinitely if the copilot CLI is not authenticated, so we rely on the earlier checkAuth to prevent that scenario.
	response, err := session.SendAndWait(ctx, copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get response from Copilot: %w", err)
	}

	if response == nil || response.Data.Content == nil {
		return "", errors.New("empty response received from Copilot")
	}

	if verboseMode {
		fmt.Fprintln(os.Stderr, "[copilot] Response received successfully")
	}

	return *response.Data.Content, nil
}

// SetVerbose enables or disables verbose mode for Copilot debug output.
func SetVerbose(v bool) {
	verboseMode = v
}

// getLogLevel returns the appropriate log level for the Copilot SDK.
func getLogLevel() string {
	if verboseMode {
		return "debug"
	}
	return "error"
}
