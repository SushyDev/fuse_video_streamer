package filesystem

import (
	"errors"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fromResponseError converts gRPC errors to standard filesystem errors
func fromResponseError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return errors.New("Unknown gRPC error")
	}

	switch st.Code() {
	case codes.NotFound:
		return syscall.ENOENT
	case codes.PermissionDenied:
		return syscall.EACCES
	case codes.AlreadyExists:
		return syscall.EEXIST
	case codes.InvalidArgument:
		return syscall.EINVAL
	case codes.ResourceExhausted:
		return syscall.ENOSPC
	case codes.FailedPrecondition:
		return syscall.EPERM
	case codes.Unimplemented:
		return syscall.ENOSYS
	default:
		return errors.New(st.Message())
	}
}
