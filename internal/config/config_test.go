package config

import (
	"os"
	"path/filepath"
	"testing"
)

// NOTE: These tests mutate environment variables and must NOT use t.Parallel().
// Each test unsets all relevant env vars at the start for isolation.

// Helper to write a temporary YAML config file for testing.
func writeTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testconfig.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}
	return tmpFile
}

// Helper to write a temporary prompt file for testing.
func writeTempPromptFile(t *testing.T, dir string) string {
	t.Helper()
	promptFile := filepath.Join(dir, "prompt.md")
	content := "Test prompt template (DIFF_CONTENT_HERE)"
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp prompt file: %v", err)
	}
	return promptFile
}

func TestLoadConfigWithOverrides_YAMLOnly(t *testing.T) {
	// Unset all relevant env vars for test isolation
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("BITBUCKET_BASE_URL")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_ENDPOINT")
	os.Unsetenv("PULLREVIEW_PROMPT_FILE")

	tmpDir := t.TempDir()
	promptFile := writeTempPromptFile(t, tmpDir)

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
  base_url: https://api.bitbucket.org/2.0
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1/chat/completions
prompt_file: ` + promptFile + `
`
	cfgFile := writeTempConfigFile(t, yaml)
	cfg, err := LoadConfigWithOverrides(cfgFile, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Bitbucket.Email != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", cfg.Bitbucket.Email)
	}
	if cfg.Bitbucket.APIToken != "token1" {
		t.Errorf("expected api_token 'token1', got '%s'", cfg.Bitbucket.APIToken)
	}
	if cfg.Bitbucket.Workspace != "ws1" {
		t.Errorf("expected workspace 'ws1', got '%s'", cfg.Bitbucket.Workspace)
	}
	if cfg.Bitbucket.BaseURL != "https://api.bitbucket.org/2.0" {
		t.Errorf("expected base_url 'https://api.bitbucket.org/2.0', got '%s'", cfg.Bitbucket.BaseURL)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", cfg.LLM.Provider)
	}
	if cfg.PromptFile != promptFile {
		t.Errorf("expected prompt_file '%s', got '%s'", promptFile, cfg.PromptFile)
	}
}

func TestLoadConfigWithOverrides_EnvOverride(t *testing.T) {
	// Unset all relevant env vars for test isolation
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("BITBUCKET_BASE_URL")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_ENDPOINT")
	os.Unsetenv("PULLREVIEW_PROMPT_FILE")

	tmpDir := t.TempDir()
	promptFile := writeTempPromptFile(t, tmpDir)

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
  base_url: https://api.bitbucket.org/2.0
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1/chat/completions
prompt_file: ` + promptFile + `
`
	cfgFile := writeTempConfigFile(t, yaml)
	os.Setenv("BITBUCKET_EMAIL", "envuser@example.com")
	os.Setenv("BITBUCKET_API_TOKEN", "envtoken")
	os.Setenv("BITBUCKET_WORKSPACE", "envws")
	os.Setenv("BITBUCKET_BASE_URL", "https://custom.bitbucket.org/api")
	os.Setenv("LLM_API_KEY", "envkey")
	defer os.Unsetenv("BITBUCKET_EMAIL")
	defer os.Unsetenv("BITBUCKET_API_TOKEN")
	defer os.Unsetenv("BITBUCKET_WORKSPACE")
	defer os.Unsetenv("BITBUCKET_BASE_URL")
	defer os.Unsetenv("LLM_API_KEY")

	cfg, err := LoadConfigWithOverrides(cfgFile, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Bitbucket.Email != "envuser@example.com" {
		t.Errorf("expected env override email 'envuser@example.com', got '%s'", cfg.Bitbucket.Email)
	}
	if cfg.Bitbucket.APIToken != "envtoken" {
		t.Errorf("expected env override api_token 'envtoken', got '%s'", cfg.Bitbucket.APIToken)
	}
	if cfg.Bitbucket.Workspace != "envws" {
		t.Errorf("expected env override workspace 'envws', got '%s'", cfg.Bitbucket.Workspace)
	}
	if cfg.Bitbucket.BaseURL != "https://custom.bitbucket.org/api" {
		t.Errorf("expected env override base_url 'https://custom.bitbucket.org/api', got '%s'", cfg.Bitbucket.BaseURL)
	}
	if cfg.LLM.APIKey != "envkey" {
		t.Errorf("expected env override api_key 'envkey', got '%s'", cfg.LLM.APIKey)
	}
}

func TestLoadConfigWithOverrides_CLIOverride(t *testing.T) {
	// Unset all relevant env vars for test isolation
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("BITBUCKET_BASE_URL")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_ENDPOINT")
	os.Unsetenv("PULLREVIEW_PROMPT_FILE")

	tmpDir := t.TempDir()
	promptFile := writeTempPromptFile(t, tmpDir)

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
  base_url: https://api.bitbucket.org/2.0
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1/chat/completions
prompt_file: ` + promptFile + `
`
	cfgFile := writeTempConfigFile(t, yaml)
	os.Setenv("BITBUCKET_EMAIL", "envuser@example.com")
	os.Setenv("BITBUCKET_API_TOKEN", "envtoken")
	os.Setenv("BITBUCKET_WORKSPACE", "envws")
	os.Setenv("BITBUCKET_BASE_URL", "https://custom.bitbucket.org/api")
	defer os.Unsetenv("BITBUCKET_EMAIL")
	defer os.Unsetenv("BITBUCKET_API_TOKEN")
	defer os.Unsetenv("BITBUCKET_WORKSPACE")
	defer os.Unsetenv("BITBUCKET_BASE_URL")

	cfg, err := LoadConfigWithOverrides(cfgFile, "cliuser@example.com", "clitoken", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Bitbucket.Email != "cliuser@example.com" {
		t.Errorf("expected CLI override email 'cliuser@example.com', got '%s'", cfg.Bitbucket.Email)
	}
	if cfg.Bitbucket.APIToken != "clitoken" {
		t.Errorf("expected CLI override api_token 'clitoken', got '%s'", cfg.Bitbucket.APIToken)
	}
	// Should still use env for workspace/base_url since CLI flags not provided for those
	if cfg.Bitbucket.Workspace != "envws" {
		t.Errorf("expected env override workspace 'envws', got '%s'", cfg.Bitbucket.Workspace)
	}
	if cfg.Bitbucket.BaseURL != "https://custom.bitbucket.org/api" {
		t.Errorf("expected env override base_url 'https://custom.bitbucket.org/api', got '%s'", cfg.Bitbucket.BaseURL)
	}
}

func TestLoadConfigWithOverrides_MissingRequired(t *testing.T) {
	// Unset all relevant env vars for test isolation
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("BITBUCKET_BASE_URL")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_ENDPOINT")
	os.Unsetenv("PULLREVIEW_PROMPT_FILE")

	yaml := `
bitbucket:
  email: ""
  api_token: ""
  workspace: ""
  base_url: ""
llm:
  provider: ""
  api_key: ""
  endpoint: ""
prompt_file: ""
`
	cfgFile := writeTempConfigFile(t, yaml)
	_, err := LoadConfigWithOverrides(cfgFile, "", "", "")
	if err == nil {
		t.Fatal("expected error for missing required config, got nil")
	}
	expected := "missing required config values"
	if err != nil && err.Error()[:len(expected)] != expected {
		t.Errorf("expected error to start with '%s', got '%v'", expected, err)
	}
}

func TestLoadConfigWithOverrides_EnvAndCLIPrecedence(t *testing.T) {
	// Unset all relevant env vars for test isolation
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("BITBUCKET_BASE_URL")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")
	os.Unsetenv("LLM_ENDPOINT")
	os.Unsetenv("PULLREVIEW_PROMPT_FILE")

	tmpDir := t.TempDir()
	promptFile := writeTempPromptFile(t, tmpDir)

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
  base_url: https://api.bitbucket.org/2.0
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1/chat/completions
prompt_file: ` + promptFile + `
`
	cfgFile := writeTempConfigFile(t, yaml)
	os.Setenv("BITBUCKET_EMAIL", "envuser@example.com")
	os.Setenv("BITBUCKET_API_TOKEN", "envtoken")
	os.Setenv("BITBUCKET_WORKSPACE", "envws")
	os.Setenv("BITBUCKET_BASE_URL", "https://custom.bitbucket.org/api")
	defer os.Unsetenv("BITBUCKET_EMAIL")
	defer os.Unsetenv("BITBUCKET_API_TOKEN")
	defer os.Unsetenv("BITBUCKET_WORKSPACE")
	defer os.Unsetenv("BITBUCKET_BASE_URL")

	cfg, err := LoadConfigWithOverrides(cfgFile, "cliuser@example.com", "clitoken", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// CLI should override env
	if cfg.Bitbucket.Email != "cliuser@example.com" {
		t.Errorf("expected CLI override email 'cliuser@example.com', got '%s'", cfg.Bitbucket.Email)
	}
	if cfg.Bitbucket.APIToken != "clitoken" {
		t.Errorf("expected CLI override api_token 'clitoken', got '%s'", cfg.Bitbucket.APIToken)
	}
	// Env should override YAML for workspace, base_url
	if cfg.Bitbucket.Workspace != "envws" {
		t.Errorf("expected env override workspace 'envws', got '%s'", cfg.Bitbucket.Workspace)
	}
	if cfg.Bitbucket.BaseURL != "https://custom.bitbucket.org/api" {
		t.Errorf("expected env override base_url 'https://custom.bitbucket.org/api', got '%s'", cfg.Bitbucket.BaseURL)
	}
}
