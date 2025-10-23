package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM          LLMConfig          `yaml:"llm"`
	Tools        ToolsConfig        `yaml:"tools"`
	Confirmation ConfirmationConfig `yaml:"confirmation"`
	Logging      LoggingConfig      `yaml:"logging"`
	BaseDir      string             `yaml:"-"` // Set at runtime to config file's directory (for logs)
	WorkingDir   string             `yaml:"-"` // Set at runtime to current working directory (for TODO.md)
}

type LLMConfig struct {
	Endpoint       string             `yaml:"endpoint"`
	APIKey         string             `yaml:"api_key"`
	Model          string             `yaml:"model"`
	Temperature    float32            `yaml:"temperature"`
	MaxTokens      int                `yaml:"max_tokens"`
	ContextWindow  int                `yaml:"context_window"`
	AutoManage     bool               `yaml:"auto_manage"`
	StartupTimeout int                `yaml:"startup_timeout"`
	Server         ServerConfig       `yaml:"server"`
}

type ServerConfig struct {
	ModelPath     string  `yaml:"model_path"`
	Host          string  `yaml:"host"`
	Port          int     `yaml:"port"`
	CtxSize       int     `yaml:"ctx_size"`
	FlashAttn     bool    `yaml:"flash_attn"`
	Jinja         bool    `yaml:"jinja"`
	CacheTypeK    string  `yaml:"cache_type_k"`
	CacheTypeV    string  `yaml:"cache_type_v"`
	BatchSize     int     `yaml:"batch_size"`
	UBatchSize    int     `yaml:"ubatch_size"`
	NCpuMoe       int     `yaml:"n_cpu_moe"`
	NGpuLayers    int     `yaml:"n_gpu_layers"`
	RepeatLastN   int     `yaml:"repeat_last_n"`
	RepeatPenalty float64 `yaml:"repeat_penalty"`
	Threads       int     `yaml:"threads"`
}

type ToolsConfig struct {
	Enabled []string `yaml:"enabled"`
}

type ConfirmationConfig struct {
	Mode               string   `yaml:"mode"`
	AutoApproveTools   []string `yaml:"auto_approve_tools"`
	AlwaysConfirmTools []string `yaml:"always_confirm_tools"`
}

type LoggingConfig struct {
	Format          string `yaml:"format"`
	Directory       string `yaml:"directory"`
	Level           string `yaml:"level"`
	LogToolResults  bool   `yaml:"log_tool_results"`
	LogReasoning    bool   `yaml:"log_reasoning"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Override with environment variables if present
	if apiKey := os.Getenv("CODER_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if endpoint := os.Getenv("CODER_ENDPOINT"); endpoint != "" {
		cfg.LLM.Endpoint = endpoint
	}

	return &cfg, nil
}
