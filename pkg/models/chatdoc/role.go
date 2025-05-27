package chatdoc

type RoleConfig struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Handler     string `yaml:"handler" json:"handler"`
	Description string `yaml:"description" json:"description"`
}
