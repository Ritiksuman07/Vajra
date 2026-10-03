package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Service ServiceConfig `mapstructure:"service"`
	Models  ModelsConfig  `mapstructure:"models"`
	Agents  AgentsConfig  `mapstructure:"agents"`
	MCP     MCPConfig     `mapstructure:"mcp"`
}

type ServiceConfig struct {
	Port     int    `mapstructure:"port"`
	DataDir  string `mapstructure:"data_dir"`
	LogLevel string `mapstructure:"log_level"`
}

type ModelsConfig struct {
	DefaultBackend string         `mapstructure:"default_backend"`
	Ollama         OllamaConfig   `mapstructure:"ollama"`
	LMStudio       LMStudioConfig `mapstructure:"lmstudio"`
	LlamaCPP       LlamaCPPConfig `mapstructure:"llama_cpp"`
}

type OllamaConfig struct {
	Host        string `mapstructure:"host"`
	DefaultModel string `mapstructure:"default_model"`
}

type LMStudioConfig struct {
	Host        string `mapstructure:"host"`
	DefaultModel string `mapstructure:"default_model"`
}

type LlamaCPPConfig struct {
	ModelPath  string `mapstructure:"model_path"`
	NThreads   int    `mapstructure:"n_threads"`
	NGPULayers int    `mapstructure:"n_gpu_layers"`
}

type AgentsConfig struct {
	MaxPods         int            `mapstructure:"max_pods"`
	DefaultResources ResourceConfig `mapstructure:"default_resources"`
}

type ResourceConfig struct {
	CPU    string `mapstructure:"cpu"`
	Memory string `mapstructure:"memory"`
}

type MCPConfig struct {
	Servers []MCPServerConfig `mapstructure:"servers"`
}

type MCPServerConfig struct {
	Name    string `mapstructure:"name"`
	Enabled bool   `mapstructure:"enabled"`
	Command string `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Default config paths
	homeDir, _ := os.UserHomeDir()
	viper.AddConfigPath(".")
	viper.AddConfigPath(filepath.Join(homeDir, ".config", "vajra-bot"))
	viper.AddConfigPath("/etc/vajra-bot")

	// Environment variable overrides
	viper.SetEnvPrefix("vajra_BOT")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found is OK, use defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("service.port", 4096)
	viper.SetDefault("service.data_dir", "~/.local/share/vajra-bot")
	viper.SetDefault("service.log_level", "info")

	viper.SetDefault("models.default_backend", "ollama")
	viper.SetDefault("models.ollama.host", "http://localhost:11434")
	viper.SetDefault("models.ollama.default_model", "llama3.1:8b")

	viper.SetDefault("models.lmstudio.host", "http://localhost:1234")
	viper.SetDefault("models.lmstudio.default_model", "llama3.1:8b")

	viper.SetDefault("agents.max_pods", 8)
	viper.SetDefault("agents.default_resources.cpu", "2")
	viper.SetDefault("agents.default_resources.memory", "4g")

	viper.SetDefault("mcp.servers", []MCPServerConfig{
		{Name: "file-system", Enabled: true},
		{Name: "shell", Enabled: true},
		{Name: "git", Enabled: true},
		{Name: "code-exec", Enabled: true},
		{Name: "knowledge-graph", Enabled: true},
	})
}
