package v1

import (
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
)

type api struct {
	sessionService SessionService
	authv1.UnimplementedAuthServiceServer
}

func NewApi(sessionService SessionService) *api {
	return &api{
		sessionService: sessionService,
	}
}
