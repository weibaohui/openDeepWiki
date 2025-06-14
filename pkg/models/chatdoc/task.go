package chatdoc

type Task struct {
	Role     string            `json:"role"`
	Type     string            `json:"type"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
	IsFinal  bool              `json:"is_final"`
	Step     string            `json:"step"`
	Inputs   []string          `json:"inputs"`
	Outputs  []string          `json:"outputs"`
}
