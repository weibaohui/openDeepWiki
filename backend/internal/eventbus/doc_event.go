package eventbus

type DocEventType string

const (
	DocEventRated  DocEventType = "Rated"
	DocEventPulled DocEventType = "DocPulled"
	DocEventPushed DocEventType = "DocPushed"
)

type DocEvent struct {
	Type         DocEventType
	RepositoryID uint
	DocID        uint
	Rating       int // 评分
}

type DocEventHandler = Handler[DocEvent]
type DocEventBus = Bus[DocEventType, DocEvent]

func NewDocEventBus() *DocEventBus {
	return NewBus[DocEventType, DocEvent]()
}
