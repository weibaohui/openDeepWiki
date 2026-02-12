package eventbus

import "github.com/weibaohui/opendeepwiki/backend/internal/domain"

type TaskEventType string

const (
	TaskEventDocWrite     TaskEventType = "DocWrite"
	TaskEventTocWrite     TaskEventType = "TocWrite"
	TaskEventTitleRewrite TaskEventType = "TitleRewrite"
	TaskEventUserRequest  TaskEventType = "UserRequest"
)

type TaskEvent struct {
	Type         TaskEventType
	RepositoryID uint
	Title        string
	SortOrder    int
	RunAfter     uint
	DocID        uint
	WriterName   domain.WriterName
}

type TaskEventHandler = Handler[TaskEvent]
type TaskEventBus = Bus[TaskEventType, TaskEvent]

func NewTaskEventBus() *TaskEventBus {
	return NewBus[TaskEventType, TaskEvent]()
}
