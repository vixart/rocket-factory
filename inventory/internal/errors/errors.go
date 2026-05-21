package errs

import "github.com/go-faster/errors"

var (
	ErrPartNotFound = errors.New("деталь не найдена")
	ErrInvalidUUID  = errors.New("неверный uuid")
)
