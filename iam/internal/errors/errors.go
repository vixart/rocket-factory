package errs

import "errors"

var (
	// Ошибки пользователей.
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrUserAlreadyExists  = errors.New("пользователь с таким логином уже существует")
	ErrInvalidCredentials = errors.New("неверный логин или пароль")

	// Ошибки сессий.
	ErrSessionNotFound = errors.New("сессия не найдена или истекла")

	// Ошибки валидации.
	ErrInvalidLogin    = errors.New("логин обязателен")
	ErrWeakPassword    = errors.New("пароль должен содержать минимум 8 символов")
	ErrEmptyCredential = errors.New("логин и пароль обязательны")
	ErrEmptySessionID  = errors.New("session_uuid обязателен")
	ErrInvalidUUID     = errors.New("неверный формат UUID")
)
