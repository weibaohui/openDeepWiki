package domain

// generationResult 表示 Agent 输出的任务生成结果（仅包内使用）。
type DirMakerGenerationResult struct {
	Dirs            []*DirMakerDirSpec `json:"dirs" yaml:"dirs"`
	AnalysisSummary string             `json:"analysis_summary" yaml:"analysis_summary"`
}

// taskSpec 表示 Agent 生成的单个任务定义（仅包内使用）。
// Type 字段不局限于预定义值，Agent 可根据项目特征自由定义。
type DirMakerDirSpec struct {
	Title     string             `json:"title" yaml:"title"`           // 目录标题，如 "安全分析"
	SortOrder int                `json:"sort_order" yaml:"sort_order"` // 排序顺序
	Hint      []DirMakerHintSpec `json:"hint" yaml:"hint"`
	DocID     uint               `json:"doc_id" yaml:"doc_id"` // 关联的文档ID 保存到数据库后才有
}

type DirMakerHintSpec struct {
	Aspect string `json:"aspect" yaml:"aspect"`
	Source string `json:"source" yaml:"source"`
	Detail string `json:"detail" yaml:"detail"`
}
