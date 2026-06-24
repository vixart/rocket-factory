package iam

import "time"

type service struct {
	userRepository UserRepository
	sessionStorage SessionRepository
	sessionTTL     time.Duration
	bcryptCost     int
}

func NewService(
	userRepository UserRepository,
	sessionStorage SessionRepository,
	sessionTTL time.Duration,
	bcryptCost int,
) *service {
	return &service{
		userRepository: userRepository,
		sessionStorage: sessionStorage,
		sessionTTL:     sessionTTL,
		bcryptCost:     bcryptCost,
	}
}
