package scope

type Scope int

const (
	Singleton Scope = iota
	Transient
	Request
	Pooled
)

func (s Scope) String() string {
	switch s {
	case Singleton:
		return "singleton"
	case Transient:
		return "transient"
	case Request:
		return "request"
	case Pooled:
		return "pooled"
	default:
		return "unknown"
	}
}
