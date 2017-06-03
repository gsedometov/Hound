package status

type Code int

const (
	Starting Code = iota
	Indexing
	Ready
	Stopping
	Error
	Undefined
)

func (c Code) String() string {
	switch c {
	case Starting:
		return "Starting"
	case Indexing:
		return "Indexing"
	case Ready:
		return "Ready"
	case Stopping:
		return "Stopping"
	case Error:
		return "Error"
	default:
		return "Undefined"
	}
}