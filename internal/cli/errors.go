package cli

import (
	"errors"
	"fmt"

	"github.com/KoukeNeko/taiga-cli/internal/output"
	"github.com/KoukeNeko/taiga-cli/internal/taiga"
	"github.com/spf13/cobra"
)

const (
	ExitSuccess              = 0
	ExitGeneric              = 1
	ExitUsage                = 2
	ExitAuth                 = 3
	ExitForbidden            = 4
	ExitNotFound             = 5
	ExitConflict             = 6
	ExitValidation           = 7
	ExitThrottled            = 8
	ExitTransport            = 9
	ExitConfirmationRequired = 10
	ExitAmbiguousCommit      = 11
)

type contractError struct {
	Code      string
	Message   string
	ExitCode  int
	Retryable bool
	Details   map[string]any
	Cause     error
}

func (e *contractError) Error() string { return e.Message }
func (e *contractError) Unwrap() error { return e.Cause }

func usageError(message string) error {
	return &contractError{Code: "usage", Message: message, ExitCode: ExitUsage}
}

func validationError(code, message string) error {
	return &contractError{Code: code, Message: message, ExitCode: ExitValidation}
}

func exactArgs(expected int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != expected {
			return usageError(fmt.Sprintf("%s accepts %d argument(s), received %d", cmd.CommandPath(), expected, len(args)))
		}
		return nil
	}
}

func authRequired(message string) error {
	return &contractError{Code: "authentication_required", Message: message, ExitCode: ExitAuth}
}

func classifyError(err error) (*contractError, output.ErrorBody) {
	var known *contractError
	if errors.As(err, &known) {
		return known, output.ErrorBody{Code: known.Code, Message: known.Message, Retryable: known.Retryable, Details: known.Details}
	}
	var fieldErr *output.FieldError
	if errors.As(err, &fieldErr) {
		known = &contractError{Code: "schema", Message: fieldErr.Error(), ExitCode: ExitUsage, Cause: err}
		return known, output.ErrorBody{Code: known.Code, Message: known.Message, Retryable: false}
	}
	var apiErr *taiga.Error
	if errors.As(err, &apiErr) {
		exitCode := ExitValidation
		switch apiErr.Kind {
		case taiga.KindAuth:
			exitCode = ExitAuth
		case taiga.KindForbidden:
			exitCode = ExitForbidden
		case taiga.KindNotFound:
			exitCode = ExitNotFound
		case taiga.KindConflict:
			exitCode = ExitConflict
		case taiga.KindThrottled:
			exitCode = ExitThrottled
		case taiga.KindTransport:
			exitCode = ExitTransport
		case taiga.KindAmbiguousCommit:
			exitCode = ExitAmbiguousCommit
		}
		known = &contractError{Code: string(apiErr.Kind), Message: apiErr.Message, ExitCode: exitCode, Retryable: apiErr.Retryable, Details: apiErr.Details, Cause: err}
		return known, output.ErrorBody{Code: string(apiErr.Kind), Message: apiErr.Message, Retryable: apiErr.Retryable, Details: apiErr.Details, UpstreamStatus: apiErr.UpstreamStatus}
	}
	known = &contractError{Code: "internal", Message: fmt.Sprintf("unexpected failure: %v", err), ExitCode: ExitGeneric, Cause: err}
	return known, output.ErrorBody{Code: known.Code, Message: known.Message, Retryable: false}
}
