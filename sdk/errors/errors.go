package errors

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
)

// define codes for MSWindows errors
const (
	WSAEADDRINUSE   syscall.Errno = 10048
	WSAECONNABORTED syscall.Errno = 10053
	WSAECONNRESET   syscall.Errno = 10054
	WSAECONNREFUSED syscall.Errno = 10061
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

/* for docs only
func isSyscallErrno(err error, errno uint) bool {
	var syscallErr *os.SyscallError
	if !errors.As(err, &syscallErr) {
		return false
	}
	var errErrno syscall.Errno
	if !errors.As(syscallErr, &errErrno) {
		return false
	}
	if errErrno == syscall.Errno(errno) {
		return true
	}
	return false
} */

// IsErrorAddressInUse verifies error
func IsErrorAddressInUse(err error) bool {
	if runtime.GOOS == "windows" {
		return errors.Is(err, WSAEADDRINUSE)
	}
	return errors.Is(err, syscall.EADDRINUSE)
}

// IsErrorConnection verifies error
func IsErrorConnection(err error) bool {
	return IsErrorConnectionAborted(err) ||
		IsErrorConnectionRefused(err) ||
		IsErrorConnectionReset(err)
}

// IsErrorConnectionAborted verifies error
func IsErrorConnectionAborted(err error) bool {
	if runtime.GOOS == "windows" {
		return errors.Is(err, WSAECONNABORTED)
	}
	return errors.Is(err, syscall.ECONNABORTED)
}

// IsErrorConnectionRefused verifies error
func IsErrorConnectionRefused(err error) bool {
	if runtime.GOOS == "windows" {
		return errors.Is(err, WSAECONNREFUSED)
	}
	return errors.Is(err, syscall.ECONNREFUSED)
}

// IsErrorConnectionReset verifies error
func IsErrorConnectionReset(err error) bool {
	if runtime.GOOS == "windows" {
		return errors.Is(err, WSAECONNRESET)
	}
	return errors.Is(err, syscall.ECONNRESET)
}
