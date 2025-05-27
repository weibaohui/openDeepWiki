package chatdoc

type WorkflowStep struct {
	From      string   `yaml:"from" json:"from"`
	To        string   `yaml:"to" json:"to"`
	Condition string   `yaml:"condition" json:"condition"`
	Metadata  []string `yaml:"metadata" json:"metadata"`
}

type WorkflowConfig struct {
	StartRole string         `yaml:"start_role" json:"start_role"`
	Steps     []WorkflowStep `yaml:"steps" json:"steps"`
}
