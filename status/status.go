package status

type Code int

//go:generate stringer -type=Code
const (
	Starting Code = iota
	Indexing
	Ready
	Stopping
	Error
	Undefined
)
