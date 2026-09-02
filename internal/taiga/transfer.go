package taiga

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// errTransferStalled is the cause a transferWatch cancels with.
var errTransferStalled = errors.New("transfer stalled")

// transferWatch abandons a transfer that stops moving. A file transfer has no
// useful overall deadline, being as long as the file and the link make it,
// but one that has moved nothing in either direction for the stall timeout is
// not going to finish, and a caller that is a script needs it to end.
type transferWatch struct {
	ctx     context.Context
	cancel  context.CancelCauseFunc
	timer   *time.Timer
	timeout time.Duration
}

// watchTransfer derives the context a transfer runs under. The caller's own
// context still says whether the operator stopped it; this one adds the stall.
func (c *Client) watchTransfer(ctx context.Context) *transferWatch {
	watched, cancel := context.WithCancelCause(ctx)
	watch := &transferWatch{ctx: watched, cancel: cancel, timeout: c.stallTimeout}
	watch.timer = time.AfterFunc(c.stallTimeout, func() { cancel(errTransferStalled) })
	return watch
}

// reader counts every read from source as progress. Both the request body the
// transport drains and the response body the caller drains go through it.
func (w *transferWatch) reader(source io.Reader) io.ReadCloser {
	return &progressReader{source: source, watch: w}
}

// stop ends the watch once the transfer's outcome is decided.
func (w *transferWatch) stop() {
	w.timer.Stop()
	w.cancel(nil)
}

// explain prefixes message with the stall when the watch is what ended the
// transfer, so that a person reads why rather than a bare cancellation.
func (w *transferWatch) explain(message string) string {
	if errors.Is(context.Cause(w.ctx), errTransferStalled) {
		return fmt.Sprintf("no data moved for %s; %s", w.timeout, message)
	}
	return message
}

type progressReader struct {
	source io.Reader
	watch  *transferWatch
}

func (r *progressReader) Read(buffer []byte) (int, error) {
	count, err := r.source.Read(buffer)
	if count > 0 {
		r.watch.timer.Reset(r.watch.timeout)
	}
	return count, err
}

// Close reaches the source when it can be closed. The transport closes a
// request body it has abandoned, and for a pipe that close is what releases
// the goroutine still writing into it.
func (r *progressReader) Close() error {
	if closer, ok := r.source.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
