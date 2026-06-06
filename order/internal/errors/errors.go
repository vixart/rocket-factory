package errs

import "github.com/go-faster/errors"

var (
	ErrInvalidUUID          = errors.New("некорректный uuid")
	ErrPaymentFailed        = errors.New("не удалось оплатить заказ")
	ErrInvalidPaymentMethod = errors.New("некорректный способ оплаты")
	ErrInvalidOrderStatus   = errors.New("заказ имеет недопустимый статус")
	ErrOrderNotFound        = errors.New("заказ не найден")
	ErrInternalError        = errors.New("внутренняя ошибка")

	ErrPartNotFound      = errors.New("деталь не найдена")
	ErrOutOfStock        = errors.New("детали нет на складе")
	ErrIncompatibleParts = errors.New("детали несовместимы")
	ErrPartTypeMismatch  = errors.New("неверный тип детали")
)
