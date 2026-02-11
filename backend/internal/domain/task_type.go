package domain

type TaskType string

var (
	Doc          TaskType = "Doc"          // 文档任务
	Toc          TaskType = "Toc"          // 目录任务
	TitleRewrite TaskType = "TitleRewrite" // 标题重写任务
)
