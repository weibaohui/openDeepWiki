package chatdoc

type RoleConfig struct {
	Name             string   `yaml:"name" json:"name"`
	Type             string   `yaml:"type" json:"type"`
	Description      string   `yaml:"description" json:"description"`
	Responsibilities []string `yaml:"responsibilities" json:"responsibilities"`
	Tasks            []string `yaml:"tasks" json:"tasks"`
	Output           []string `yaml:"output" json:"output"`
	Prompt           string   `yaml:"prompt" json:"prompt"`
}
