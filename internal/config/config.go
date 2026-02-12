package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"pullreview/internal/utils"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the pullreview tool.
type Config struct {
	Bitbucket struct {
		Email string `yaml:"email"` // Bitbucket Cloud account email

		APIToken string `yaml:"api_token"` // Bitbucket Cloud API token

		Workspace string `yaml:"workspace"` // Bitbucket Cloud workspace

		RepoSlug string `yaml:"repo_slug"` // Bitbucket repository slug (inferred from git if missing)
		BaseURL  string `yaml:"base_url"`  // Bitbucket API base URL (optional, defaults to https://api.bitbucket.org/2.0)

	} `yaml:"bitbucket"`

	LLM struct {
		Provider string `yaml:"provider"` // LLM provider name (e.g., openai)

		APIKey string `yaml:"api_key"` // LLM API key

		Endpoint string `yaml:"endpoint"` // LLM API endpoint

		Model string `yaml:"model"` // LLM model name (e.g., arcee-ai/trinity-large-preview:free)

	} `yaml:"llm"`

	PromptFile string `yaml:"prompt_file"` // Path to the prompt template file

	// AutoFix configuration
	AutoFix struct {
		Enabled               bool   `yaml:"enabled"`
		AutoCreatePR          bool   `yaml:"auto_create_pr"`
		MaxIterations         int    `yaml:"max_iterations"`
		VerifyBuild           bool   `yaml:"verify_build"`
		VerifyTests           bool   `yaml:"verify_tests"`
		VerifyLint            bool   `yaml:"verify_lint"`
		PipelineMode          bool   `yaml:"pipeline_mode"`
		BranchPrefix          string `yaml:"branch_prefix"`
		FixPromptFile         string `yaml:"fix_prompt_file"`
		CommitMessageTemplate string `yaml:"commit_message_template"`
		PRTitleTemplate       string `yaml:"pr_title_template"`
		PRDescriptionTemplate string `yaml:"pr_description_template"`
	} `yaml:"autofix"`
}

// LoadConfigWithOverrides loads configuration from a YAML file, then applies overrides from
// environment variables and finally from CLI flags (email, apiToken).

// Returns a validated Config or an error if required fields are missing.
func LoadConfigWithOverrides(cfgFile, email, apiToken string) (*Config, error) {

	cfg := &Config{}

	// 1. Load from YAML file
	if cfgFile == "" {
		return nil, errors.New("config file path must be provided explicitly")
	}
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", cfgFile, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not parse YAML config: %w", err)
	}

	// 2. Override with environment variables if set (but only if not set by CLI flags)
	if v := os.Getenv("BITBUCKET_EMAIL"); v != "" && email == "" {
		cfg.Bitbucket.Email = v
	}
	if v := os.Getenv("BITBUCKET_API_TOKEN"); v != "" && apiToken == "" {
		cfg.Bitbucket.APIToken = v
	}

	if v := os.Getenv("BITBUCKET_WORKSPACE"); v != "" {

		cfg.Bitbucket.Workspace = v

	}

	if v := os.Getenv("BITBUCKET_REPO_SLUG"); v != "" {
		cfg.Bitbucket.RepoSlug = v
	}
	if v := os.Getenv("BITBUCKET_BASE_URL"); v != "" {
		cfg.Bitbucket.BaseURL = v

	}

	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("LLM_ENDPOINT"); v != "" {
		cfg.LLM.Endpoint = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("PULLREVIEW_PROMPT_FILE"); v != "" {
		cfg.PromptFile = v
	}

	// 3. Override with CLI flags if provided (highest precedence)
	if email != "" {
		cfg.Bitbucket.Email = email
	}
	if apiToken != "" {
		cfg.Bitbucket.APIToken = apiToken
	}

	// 4. Set default for BaseURL if not set

	if strings.TrimSpace(cfg.Bitbucket.BaseURL) == "" {

		cfg.Bitbucket.BaseURL = "https://api.bitbucket.org/2.0"

	}

	// 4b. Infer RepoSlug from git if not set
	if strings.TrimSpace(cfg.Bitbucket.RepoSlug) == "" {
		repoPath, err := os.Getwd()
		if err == nil {
			if slug, err := inferRepoSlug(repoPath); err == nil && slug != "" {
				cfg.Bitbucket.RepoSlug = slug
			}
		}
	}

	// 5. Set defaults for Copilot provider
	if strings.ToLower(cfg.LLM.Provider) == "copilot" {
		if strings.TrimSpace(cfg.LLM.Model) == "" {
			cfg.LLM.Model = "gpt-4.1" // Default model for Copilot
		}
	}

	// 6. Validate required fields
	var missing []string
	if strings.TrimSpace(cfg.Bitbucket.Email) == "" {
		missing = append(missing, "bitbucket.email")
	}
	if strings.TrimSpace(cfg.Bitbucket.APIToken) == "" {
		missing = append(missing, "bitbucket.api_token")
	}

	if strings.TrimSpace(cfg.Bitbucket.Workspace) == "" {
		missing = append(missing, "bitbucket.workspace")
	}

	if strings.TrimSpace(cfg.Bitbucket.RepoSlug) == "" {
		missing = append(missing, "bitbucket.repo_slug (could not infer from git remote)")
	}
	if strings.TrimSpace(cfg.LLM.Provider) == "" {
		missing = append(missing, "llm.provider")
	}
	// API key is only required for non-Copilot providers
	if strings.ToLower(cfg.LLM.Provider) != "copilot" && strings.TrimSpace(cfg.LLM.APIKey) == "" {
		missing = append(missing, "llm.api_key")
	}

	if strings.TrimSpace(cfg.PromptFile) == "" {
		missing = append(missing, "prompt_file")
	}

	if len(missing) > 0 {

		return nil, errors.New("missing required config values: " + strings.Join(missing, ", "))

	}

	return cfg, nil

}

// inferRepoSlug tries to infer the Bitbucket repo slug from the git remote URL.
func inferRepoSlug(repoPath string) (string, error) {
	return utils.GetRepoSlugFromGitRemote(repoPath)
}

// DetectPipelineMode checks environment variables to determine if running in CI/CD.
func DetectPipelineMode() bool {
	ciEnvVars := []string{
		"CI",                 // Generic CI indicator
		"BITBUCKET_PIPELINE", // Bitbucket Pipelines
		"GITHUB_ACTIONS",     // GitHub Actions
		"GITLAB_CI",          // GitLab CI
		"JENKINS_HOME",       // Jenkins
		"CIRCLECI",           // CircleCI
		"TRAVIS",             // Travis CI
		"AZURE_PIPELINES",    // Azure Pipelines
		"BUDDY_WORKSPACE_ID", // Buddy
		"TEAMCITY_VERSION",   // TeamCity
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}
