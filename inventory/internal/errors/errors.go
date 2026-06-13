package errs

import "github.com/go-faster/errors"

var (
	ErrPartNotFound      = errors.New("деталь не найдена")
	ErrInvalidUUID       = errors.New("неверный uuid")
	ErrOutOfStock        = errors.New("детали нет на складе")
	ErrNothingToRelease  = errors.New("нет детали в резерве")
	ErrNothingToCommit   = errors.New("невозможно использовать деталь")
	ErrNoPartWereUpdated = errors.New("ни одна деталь не была обновлена")
	ErrIncompatibleParts = errors.New("детали несовместимы")
	ErrPartTypeMismatch  = errors.New("деталь не совпала")
	ErrInvalidProperties = errors.New("неверное свойство")
)
