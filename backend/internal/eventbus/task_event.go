package eventbus

import "github.com/weibaohui/opendeepwiki/backend/internal/domain"

type TaskEventType string

const (
	TaskEventDocWrite              TaskEventType = "DocWrite"              // 文档写入
	TaskEventTocWrite              TaskEventType = "TocWrite"              // 目录写入
	TaskEventTitleRewrite          TaskEventType = "TitleRewrite"          // 标题重写
	TaskEventUserRequest           TaskEventType = "UserRequest"           // 用户请求
	TaskEventWriteComplete         TaskEventType = "WriteComplete"         // 文档写入完成
	TaskEventWriteFailed           TaskEventType = "WriteFailed"           // 文档写入失败
	TaskEventRegenerate            TaskEventType = "Regenerate"            // 文档重新生成  //TODO 做成一个功能
	TaskEventAgentDiscussionRating TaskEventType = "AgentDiscussionRating" // 智能体讨论文章质量 //TODO 待实现
)

type TaskEvent struct {
	Type         TaskEventType
	RepositoryID uint
	Title        string
	SortOrder    int
	RunAfter     uint
	DocID        uint
	WriterName   domain.WriterName
	TaskID       uint            // 任务ID,若有
	TaskType     domain.TaskType // 任务类型
}

type TaskEventHandler = Handler[TaskEvent]
type TaskEventBus = Bus[TaskEventType, TaskEvent]

func NewTaskEventBus() *TaskEventBus {
	return NewBus[TaskEventType, TaskEvent]()
}
