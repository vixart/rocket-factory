package converter

import (
	"github.com/vixart/rocket-factory/iam/internal/service/input"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
)

func RegisterRequestToRegisterInput(req *userv1.RegisterRequest) input.UserRegisterInput {
	return input.UserRegisterInput{
		Login:    req.GetInfo().GetInfo().GetLogin(),
		Password: req.GetInfo().GetPassword(),
	}
}

func LoginRequestToLoginInput(req *authv1.LoginRequest) input.UserLoginInput {
	return input.UserLoginInput{
		Login:    req.GetLogin(),
		Password: req.GetPassword(),
	}
}
