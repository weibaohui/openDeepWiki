package chatdoc

type RoleConfig struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Prompt      string `yaml:"prompt" json:"prompt"`
}
