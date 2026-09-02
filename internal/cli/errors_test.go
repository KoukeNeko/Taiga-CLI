package cli

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KoukeNeko/aihki/internal/taiga"
)

// An interrupted write reports that something may have been committed. The
// cancellation that caused it is still reachable through the error chain, so
// the order these are tested in decides whether that warning survives.
func TestClassifyErrorKeepsAnUnconfirmedWriteAheadOfItsCancellation(t *testing.T) {
	unconfirmed := &taiga.Error{
		Kind:    taiga.KindAmbiguousCommit,
		Message: "request was interrupted before Taiga confirmed it; verify before retrying",
		Cause:   context.Canceled,
	}
	if !errors.Is(unconfirmed, context.Canceled) {
		t.Fatal("this test is pointless unless the error really does wrap the cancellation")
	}
	known, body := classifyError(unconfirmed)
	if known.ExitCode != ExitAmbiguousCommit {
		t.Errorf("exit code = %d, want %d", known.ExitCode, ExitAmbiguousCommit)
	}
	if body.Code != string(taiga.KindAmbiguousCommit) {
		t.Errorf("code = %q, want %q", body.Code, taiga.KindAmbiguousCommit)
	}
}

// A cancellation carrying nothing else is the operator's own decision, not a
// defect in the command.
func TestClassifyErrorReportsAPlainInterruptAsSuch(t *testing.T) {
	for _, testCase := range []struct {
		err  error
		code string
	}{
		{context.Canceled, "interrupted"},
		{context.DeadlineExceeded, "timeout"},
		{fmt.Errorf("fetching issues: %w", context.Canceled), "interrupted"},
	} {
		known, body := classifyError(testCase.err)
		if known.ExitCode != ExitInterrupted {
			t.Errorf("%v: exit code = %d, want %d", testCase.err, known.ExitCode, ExitInterrupted)
		}
		if body.Code != testCase.code {
			t.Errorf("%v: code = %q, want %q", testCase.err, body.Code, testCase.code)
		}
	}
}
