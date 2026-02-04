package copilot

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedModel string
	}{
		{
			name:          "default model when empty",
			model:         "",
			expectedModel: "gpt-4.1",
		},
		{
			name:          "custom model",
			model:         "gpt-5",
			expectedModel: "gpt-5",
		},
		{
			name:          "another custom model",
			model:         "claude-sonnet-4.5",
			expectedModel: "claude-sonnet-4.5",
		},
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
		})
	}
}

func TestCheckCLIAvailable(t *testing.T) {
	// This test will fail if Copilot CLI is not installed, which is expected
	// We just verify the function doesn't panic
	err := CheckCLIAvailable()
	// We don't assert on the result since it depends on environment
	_ = err
}

func TestSetVerbose(t *testing.T) {
	// Test that SetVerbose doesn't panic
	SetVerbose(true)
	if !verboseMode {
		t.Error("SetVerbose(true) should set verboseMode to true")
	}
	SetVerbose(false)
	if verboseMode {
		t.Error("SetVerbose(false) should set verboseMode to false")
	}
}

func TestGetLogLevel(t *testing.T) {
	SetVerbose(false)
	if level := getLogLevel(); level != "error" {
		t.Errorf("getLogLevel() with verbose=false = %q, want %q", level, "error")
	}

	SetVerbose(true)
	if level := getLogLevel(); level != "debug" {
		t.Errorf("getLogLevel() with verbose=true = %q, want %q", level, "debug")
	}

	// Reset
	SetVerbose(false)
}
