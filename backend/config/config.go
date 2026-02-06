package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Data     DataConfig     `yaml:"data"`
	Agent    AgentConfig    `yaml:"agent"`
	Skill    SkillConfig    `yaml:"skill"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release
}

type DatabaseConfig struct {
	Type string `yaml:"type"` // sqlite, mysql
	DSN  string `yaml:"dsn"`
}

type LLMConfig struct {
	APIURL    string `yaml:"api_url"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

type DataConfig struct {
	Dir     string `yaml:"dir"`
	RepoDir string `yaml:"repo_dir"`
}
type AgentConfig struct {
	Dir            string
	ReloadInterval time.Duration
}
type SkillConfig struct {
	Dir string
}

var (
	cfg  *Config
	once sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		cfg = loadConfig()
	})
	return cfg
}

func loadConfig() *Config {
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Type: "sqlite",
			DSN:  "./data/app.db",
		},
		LLM: LLMConfig{
			APIURL:    "https://api.openai.com/v1",
			Model:     "gpt-4o",
			MaxTokens: 4096,
		},
		Data: DataConfig{
			Dir:     "./data",
			RepoDir: "./data/repos",
		},
		Agent: AgentConfig{
			Dir:            "./agents",
			ReloadInterval: 5 * time.Second,
		},
		Skill: SkillConfig{
			Dir: "./skills",
		},
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err == nil {
		yaml.Unmarshal(data, config)
	}

	// 环境变量优先级高于配置文件
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		config.LLM.APIURL = baseURL
	}
	if model := os.Getenv("OPENAI_MODEL_NAME"); model != "" {
		config.LLM.Model = model
	}

	// 数据库环境变量
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		config.Database.Type = dbType
	}
	if dbDSN := os.Getenv("DB_DSN"); dbDSN != "" {
		config.Database.DSN = dbDSN
	}

	// 数据目录环境变量
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		config.Data.Dir = dataDir
	}
	if repoDir := os.Getenv("REPO_DIR"); repoDir != "" {
		config.Data.RepoDir = repoDir
	}

	if config.Data.RepoDir == "" {
		config.Data.RepoDir = filepath.Join(config.Data.Dir, "repos")
	}

	if agentDir := os.Getenv("AGENT_DIR"); agentDir != "" {
		config.Agent.Dir = agentDir
	}
	if skillDir := os.Getenv("SKILL_DIR"); skillDir != "" {
		config.Skill.Dir = skillDir
	}

	return config
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func UpdateConfig(newCfg *Config) {
	cfg = newCfg
}
