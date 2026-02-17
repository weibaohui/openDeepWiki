package eventbus

type RepositoryEventType string

const (
	RepositoryEventAdded              RepositoryEventType = "Added"
	RepositoryEventDeleted            RepositoryEventType = "Deleted"
	RepositoryEventIncrementalUpdated RepositoryEventType = "IncrementalUpdated"
)

type RepositoryEvent struct {
	Type         RepositoryEventType
	RepositoryID uint
	CloneBranch  string
	CloneCommit  string
}

type RepositoryEventHandler = Handler[RepositoryEvent]
type RepositoryEventBus = Bus[RepositoryEventType, RepositoryEvent]

func NewRepositoryEventBus() *RepositoryEventBus {
	return NewBus[RepositoryEventType, RepositoryEvent]()
}
