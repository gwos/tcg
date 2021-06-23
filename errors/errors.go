package errors

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// define error types for retry logic
var (
	ErrPermanent = errors.New("permanent error")
	ErrTransient = errors.New("transient error")

	ErrGateway      = fmt.Errorf("%w: %v", ErrTransient, "gateway error")
	ErrSynchronizer = fmt.Errorf("%w: %v", ErrTransient, "synchronizer error")
	ErrUnauthorized = fmt.Errorf("%w: %v", ErrPermanent, "unauthorized")
	ErrUndecided    = fmt.Errorf("%w: %v", ErrPermanent, "undecided error")
)

// IsErrorAddressInUse verifies error
func IsErrorAddressInUse(err error) bool {
	var eOsSyscall *os.SyscallError
	if !errors.As(err, &eOsSyscall) {
		return false
	}
	var errErrno syscall.Errno
	if !errors.As(eOsSyscall, &errErrno) {
		return false
	}
	if errErrno == syscall.EADDRINUSE {
		return true
	}
	const WSAEADDRINUSE = 10048
	if runtime.GOOS == "windows" && errErrno == WSAEADDRINUSE {
		return true
	}
	return false
}
