package eventbus

type DocEventType string

const (
	DocEventRated    DocEventType = "Rated"
	DocEventPulled   DocEventType = "DocPulled"
	DocEventPushed   DocEventType = "DocPushed"
	DocEventSaved    DocEventType = "Saved"    // 文档保存事件
	DocEventUpdated  DocEventType = "Updated"  // 文档更新事件
)

type DocEvent struct {
	Type         DocEventType
	RepositoryID uint
	DocID        uint
	Rating       int // 评分
	TargetServer string
	Success      bool
	Title        string // 文档标题
	Content      string // 文档内容
}

type DocEventHandler = Handler[DocEvent]
type DocEventBus = Bus[DocEventType, DocEvent]

func NewDocEventBus() *DocEventBus {
	return NewBus[DocEventType, DocEvent]()
}
