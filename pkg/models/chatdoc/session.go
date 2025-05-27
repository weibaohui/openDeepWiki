package chatdoc

type Role struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Prompt      string `yaml:"prompt" json:"prompt"`
}

type Collaboration struct {
	From   string `yaml:"from" json:"from"`
	To     string `yaml:"to" json:"to"`
	Action string `yaml:"action" json:"action"`
}

type ChatDocSession struct {
	ID           string   `json:"id"`
	CurrentStage string   `json:"current_stage"`
	History      []string `json:"history"`
	Roles        []Role   `json:"roles"`
}
