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
prompt_file: prompt.md
`
	cfgFile := writeTempConfigFile(t, yaml)
	cfg, err := LoadConfigWithOverrides(cfgFile, "", "")
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
	if cfg.PromptFile != "prompt.md" {
		t.Errorf("expected prompt_file 'prompt.md', got '%s'", cfg.PromptFile)
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
prompt_file: prompt.md
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

	cfg, err := LoadConfigWithOverrides(cfgFile, "", "")
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
prompt_file: prompt.md
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

	cfg, err := LoadConfigWithOverrides(cfgFile, "cliuser@example.com", "clitoken")
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
	_, err := LoadConfigWithOverrides(cfgFile, "", "")
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
prompt_file: prompt.md
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

	cfg, err := LoadConfigWithOverrides(cfgFile, "cliuser@example.com", "clitoken")
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

// TestDetectPipelineMode tests pipeline mode detection.
func TestDetectPipelineMode(t *testing.T) {
	// Clear all CI env vars
	ciEnvVars := []string{
		"CI", "BITBUCKET_PIPELINE", "GITHUB_ACTIONS", "GITLAB_CI",
		"JENKINS_HOME", "CIRCLECI", "TRAVIS", "AZURE_PIPELINES",
		"BUDDY_WORKSPACE_ID", "TEAMCITY_VERSION",
	}
	for _, env := range ciEnvVars {
		os.Unsetenv(env)
	}

	// Not in pipeline mode
	if DetectPipelineMode() {
		t.Error("expected DetectPipelineMode()=false with no CI env vars")
	}

	// Set CI env var
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	if !DetectPipelineMode() {
		t.Error("expected DetectPipelineMode()=true with CI=true")
	}
}

// TestAutoFixConfig tests AutoFix configuration loading.
func TestAutoFixConfig(t *testing.T) {
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1
prompt_file: prompt.md
autofix:
  enabled: true
  auto_create_pr: true
  max_iterations: 10
  verify_build: true
  verify_tests: true
  verify_lint: false
  pipeline_mode: false
  branch_prefix: fix-branch
  fix_prompt_file: prompts/fix.md
  commit_message_template: "Fix: {issue_summary}"
  pr_title_template: "Auto-fix PR #{pr_id}"
  pr_description_template: "Fixed issues"
`
	cfgFile := writeTempConfigFile(t, yaml)
	cfg, err := LoadConfigWithOverrides(cfgFile, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.AutoFix.Enabled {
		t.Error("expected autofix.enabled=true")
	}
	if !cfg.AutoFix.AutoCreatePR {
		t.Error("expected autofix.auto_create_pr=true")
	}
	if cfg.AutoFix.MaxIterations != 10 {
		t.Errorf("expected autofix.max_iterations=10, got %d", cfg.AutoFix.MaxIterations)
	}
	if !cfg.AutoFix.VerifyBuild {
		t.Error("expected autofix.verify_build=true")
	}
	if !cfg.AutoFix.VerifyTests {
		t.Error("expected autofix.verify_tests=true")
	}
	if cfg.AutoFix.VerifyLint {
		t.Error("expected autofix.verify_lint=false")
	}
	if cfg.AutoFix.BranchPrefix != "fix-branch" {
		t.Errorf("expected autofix.branch_prefix=fix-branch, got %q", cfg.AutoFix.BranchPrefix)
	}
	if cfg.AutoFix.FixPromptFile != "prompts/fix.md" {
		t.Errorf("expected autofix.fix_prompt_file=prompts/fix.md, got %q", cfg.AutoFix.FixPromptFile)
	}
	if cfg.AutoFix.CommitMessageTemplate != "Fix: {issue_summary}" {
		t.Errorf("expected commit template, got %q", cfg.AutoFix.CommitMessageTemplate)
	}
	if cfg.AutoFix.PRTitleTemplate != "Auto-fix PR #{pr_id}" {
		t.Errorf("expected PR title template, got %q", cfg.AutoFix.PRTitleTemplate)
	}
	if cfg.AutoFix.PRDescriptionTemplate != "Fixed issues" {
		t.Errorf("expected PR description template, got %q", cfg.AutoFix.PRDescriptionTemplate)
	}
}

// TestAutoFixConfigDefaults tests that missing autofix config doesn't break loading.
func TestAutoFixConfigDefaults(t *testing.T) {
	os.Unsetenv("BITBUCKET_EMAIL")
	os.Unsetenv("BITBUCKET_API_TOKEN")
	os.Unsetenv("BITBUCKET_WORKSPACE")
	os.Unsetenv("LLM_PROVIDER")
	os.Unsetenv("LLM_API_KEY")

	yaml := `
bitbucket:
  email: user@example.com
  api_token: token1
  workspace: ws1
llm:
  provider: openai
  api_key: key1
  endpoint: https://api.openai.com/v1
prompt_file: prompt.md
`
	cfgFile := writeTempConfigFile(t, yaml)
	cfg, err := LoadConfigWithOverrides(cfgFile, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AutoFix section should exist but be empty/defaults
	if cfg.AutoFix.Enabled {
		t.Error("expected autofix.enabled=false by default")
	}
	if cfg.AutoFix.MaxIterations != 0 {
		t.Errorf("expected autofix.max_iterations=0 by default, got %d", cfg.AutoFix.MaxIterations)
	}
}
