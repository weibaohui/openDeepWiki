package domain

type TaskType string

var (
	DocWrite     TaskType = "DocWrite"     // 文档任务
	TocWrite     TaskType = "TocWrite"     // 目录任务
	TitleRewrite TaskType = "TitleRewrite" // 标题重写任务
)
