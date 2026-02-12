package eventbus

type RepositoryEventType string

const (
	RepositoryEventAdded   RepositoryEventType = "Added"
	RepositoryEventDeleted RepositoryEventType = "Deleted"
)

type RepositoryEvent struct {
	Type         RepositoryEventType
	RepositoryID uint
}

type RepositoryEventHandler = Handler[RepositoryEvent]
type RepositoryEventBus = Bus[RepositoryEventType, RepositoryEvent]

func NewRepositoryEventBus() *RepositoryEventBus {
	return NewBus[RepositoryEventType, RepositoryEvent]()
}
