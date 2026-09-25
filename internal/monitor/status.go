package monitor

type CollectorState string

const (
	CollectorAvailable    CollectorState = "available"
	CollectorUnavailable  CollectorState = "unavailable"
	CollectorNotSupported CollectorState = "not_supported"
	CollectorUnknown      CollectorState = "unknown"
)

type CollectorStatus struct {
	Required bool
	State    CollectorState
}
