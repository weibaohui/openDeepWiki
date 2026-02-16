package domain

type TaskType string

var (
	DocWrite         TaskType = "DocWrite"         // 文档任务
	DocRewrite       TaskType = "DocRewrite"       // 文档重写任务，比如更新、删除、修改文档中的部分内容。
	TocWrite         TaskType = "TocWrite"         // 目录任务
	TitleRewrite     TaskType = "TitleRewrite"     // 标题重写任务
	IncrementalWrite TaskType = "IncrementalWrite" // 增量更新任务
)
