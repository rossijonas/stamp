package cli

import (
	"errors"
	"fmt"
)

// Error categories classify failures by kind so the exit boundary can map
// them to sysexits codes without commands knowing about exit numbers.
// Match the existing ErrNotSupported sentinel pattern in internal/manager.
var (
	// ErrUsage marks a bad command-line argument or flag value.
	ErrUsage = errors.New("usage error")
	// ErrData marks malformed or ambiguous input data.
	ErrData = errors.New("data error")
	// ErrNoInput marks a referenced input file or value that does not exist.
	ErrNoInput = errors.New("input missing")
	// ErrUnavailable marks a required resource (e.g. a package manager) that is
	// not present on this system.
	ErrUnavailable = errors.New("unavailable")
	// ErrCanTCreate marks a failure to create or write an output file.
	ErrCanTCreate = errors.New("cannot create")
	// ErrConfig marks an unconfigured or misconfigured state.
	ErrConfig = errors.New("configuration error")
)

// Exit codes following BSD sysexits.h conventions (glibc ships it on Linux).
const (
	ExitUsage       = 64 // EX_USAGE: command line usage error
	ExitDataErr     = 65 // EX_DATAERR: data format incorrect
	ExitNoInput     = 66 // EX_NOINPUT: cannot open input
	ExitUnavailable = 69 // EX_UNAVAILABLE: service unavailable
	ExitSoftware    = 70 // EX_SOFTWARE: internal software error
	ExitCanTCreate  = 73 // EX_CANTCREAT: cannot create output file
	ExitConfig      = 78 // EX_CONFIG: configuration error
)

// categorizedError attaches a category sentinel to an existing error without
// altering its message, so user-facing text and string assertions are
// unchanged while errors.Is can classify the chain.
type categorizedError struct {
	category error
	err      error
}

func (e *categorizedError) Error() string { return e.err.Error() }
func (e *categorizedError) Unwrap() error { return e.err }

// Is matches the category sentinel by identity.
func (e *categorizedError) Is(target error) bool { return target == e.category }

// catErr builds a categorized error with the given message.
func catErr(category error, format string, args ...any) error {
	return &categorizedError{category: category, err: fmt.Errorf(format, args...)}
}

// exitCodeFor maps an error to its sysexits exit code. The default is 1, the
// POSIX catchall: only explicitly categorized errors get distinct codes.
// errors.Is traverses wrapped errors and errors.Join members.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrUsage):
		return ExitUsage
	case errors.Is(err, ErrData):
		return ExitDataErr
	case errors.Is(err, ErrNoInput):
		return ExitNoInput
	case errors.Is(err, ErrUnavailable):
		return ExitUnavailable
	case errors.Is(err, ErrCanTCreate):
		return ExitCanTCreate
	case errors.Is(err, ErrConfig):
		return ExitConfig
	}
	return 1
}
