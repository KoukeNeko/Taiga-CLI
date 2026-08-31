package taiga

import "fmt"

type ErrorKind string

const (
	KindAuth            ErrorKind = "auth"
	KindForbidden       ErrorKind = "forbidden"
	KindNotFound        ErrorKind = "not_found"
	KindConflict        ErrorKind = "occ_conflict"
	KindValidation      ErrorKind = "validation"
	KindThrottled       ErrorKind = "throttled"
	KindTransport       ErrorKind = "transport"
	KindAmbiguousCommit ErrorKind = "ambiguous_commit"
)

type Error struct {
	Kind           ErrorKind      `json:"code"`
	Operation      string         `json:"-"`
	Message        string         `json:"message"`
	Retryable      bool           `json:"retryable"`
	UpstreamStatus int            `json:"upstream_status,omitempty"`
	Details        map[string]any `json:"details,omitempty"`
	Cause          error          `json:"-"`
}

func (e *Error) Error() string {
	if e.Operation == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Operation, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }
