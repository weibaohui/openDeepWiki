package chatdoc

type WorkflowStep struct {
	Step     string         `yaml:"step" json:"step"`
	Actor    string         `yaml:"actor" json:"actor"`
	Input    []string       `yaml:"input" json:"input"`
	Output   []string       `yaml:"output" json:"output"`
	Substeps []WorkflowStep `yaml:"substeps" json:"substeps"`
}

type WorkflowConfig struct {
	StartRole string         `yaml:"start_role" json:"start_role"`
	Steps     []WorkflowStep `yaml:"steps" json:"steps"`
}
