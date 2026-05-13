package claudecode

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
			expectedModel: "sonnet",
		},
		{
			name:          "default model when whitespace",
			model:         "   ",
			expectedModel: "sonnet",
		},
		{
			name:          "custom alias",
			model:         "opus",
			expectedModel: "opus",
		},
		{
			name:          "explicit model id",
			model:         "claude-sonnet-4-6",
			expectedModel: "claude-sonnet-4-6",
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
	// Result depends on the environment (claude installed + logged in).
	// We just verify the function doesn't panic.
	_ = CheckCLIAvailable()
}

func TestSetVerbose(t *testing.T) {
	SetVerbose(true)
	if !verboseMode {
		t.Error("SetVerbose(true) should set verboseMode to true")
	}
	SetVerbose(false)
	if verboseMode {
		t.Error("SetVerbose(false) should set verboseMode to false")
	}
}
