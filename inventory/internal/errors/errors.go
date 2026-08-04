package errs

import "github.com/go-faster/errors"

var (
	ErrPartNotFound      = errors.New("part not found")
	ErrInvalidUUID       = errors.New("invalid uuid")
	ErrOutOfStock        = errors.New("part is out of stock")
	ErrNothingToRelease  = errors.New("no reserved part to release")
	ErrNothingToCommit   = errors.New("part cannot be committed")
	ErrNoPartWereUpdated = errors.New("no part was updated")
	ErrIncompatibleParts = errors.New("parts are incompatible")
	ErrPartTypeMismatch  = errors.New("part type mismatch")
	ErrInvalidProperties = errors.New("invalid property")
	ErrInternalError     = errors.New("internal error")
	ErrUnauthenticated   = errors.New("authentication failed")
)
