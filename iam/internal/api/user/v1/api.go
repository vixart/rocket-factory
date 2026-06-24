package v1

import (
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

type api struct {
	userService UserService
	userv1.UnimplementedUserServiceServer
}

func NewApi(userService UserService) *api {
	return &api{
		userService: userService,
	}
}
