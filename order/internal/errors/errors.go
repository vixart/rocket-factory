package errs

import "github.com/go-faster/errors"

var (
	ErrInventoryPartNotFound = errors.New("деталь не найдена")
	ErrInvalidUUID           = errors.New("некорректный uuid")
	ErrPaymentFailed         = errors.New("не удалось оплатить заказ")
	ErrInvalidPaymentMethod  = errors.New("указан недопустимый метод оплаты")
	ErrInvalidOrderStatus    = errors.New("заказ имеет недопустимый статус")
	ErrOrderAlreadyExists    = errors.New("заказ с таким uuid уже существует")
	ErrOrderNotFound         = errors.New("заказ не найден")
	ErrPartInsufficientStock = errors.New("детали нет на складе")
	ErrPartNotFound          = errors.New("деталь не найдена")
	ErrInternalError         = errors.New("внутренняя ошибка")
)
