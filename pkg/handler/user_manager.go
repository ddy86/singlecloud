package handler

import (
	"fmt"
	"time"

	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/authorize"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserManager struct {
	api.DefaultHandler
	impl *authorize.UserManager
}

func newUserManager(secret []byte, tokenValidDuration time.Duration) *UserManager {
	return &UserManager{
		impl: authorize.NewUserManager(secret, tokenValidDuration),
	}
}

func (m *UserManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	user := ctx.Object.(*types.User)
	if err := m.impl.AddUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, "duplicate user name")
	}
	user.SetID(user.Name)
	user.SetType(types.UserType)
	user.SetCreationTimestamp(time.Now())
	return hideUserPassword(user), nil
}

func (m *UserManager) Get(ctx *resttypes.Context) interface{} {
	user := m.impl.GetUser(ctx.Object.GetID())
	if user != nil {
		return hideUserPassword(user)
	} else {
		return nil
	}
}

func (m *UserManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if err := m.impl.DeleteUser(ctx.Object.GetID()); err != nil {
		return resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return nil
}

func (m *UserManager) Update(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	user := ctx.Object.(*types.User)
	if err := m.impl.UpdateUser(user); err != nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, err.Error())
	}
	return hideUserPassword(user), nil
}

func (m *UserManager) List(ctx *resttypes.Context) interface{} {
	users := m.impl.GetUsers()
	var ret []*types.User
	for _, user := range users {
		ret = append(ret, hideUserPassword(user))
	}
	return ret
}

func (m *UserManager) Action(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	switch ctx.Action.Name {
	case types.ActionLogin:
		return m.login(ctx)
	case types.ActionResetPassword:
		return nil, m.resetPassword(ctx)
	default:
		return nil, resttypes.NewAPIError(resttypes.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Action.Name))
	}
}

func (m *UserManager) login(ctx *resttypes.Context) (interface{}, *resttypes.APIError) {
	up, ok := ctx.Action.Input.(*types.UserPassword)
	if ok == false {
		return nil, resttypes.NewAPIError(resttypes.InvalidFormat, "login param not valid")
	}

	token, err := m.impl.CreateToken(ctx.Object.GetID(), up.Password)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	} else {
		return map[string]string{
			"token": token,
		}, nil
	}
}

func (m *UserManager) resetPassword(ctx *resttypes.Context) *resttypes.APIError {
	param, ok := ctx.Action.Input.(*types.ResetPassword)
	if ok == false {
		return resttypes.NewAPIError(resttypes.InvalidFormat, "reset password param not valid")
	}

	user := getCurrentUser(ctx)
	if user.Name != ctx.Object.GetID() {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only user himself could reset his password")
	}

	err := m.impl.ResetPassword(user.Name, param.OldPassword, param.NewPassword)
	if err != nil {
		return resttypes.NewAPIError(resttypes.PermissionDenied, err.Error())
	} else {
		return nil
	}
}

func (m *UserManager) createAuthenticationHandler() api.HandlerFunc {
	return func(ctx *resttypes.Context) *resttypes.APIError {
		return m.impl.HandleRequest(ctx)
	}
}

func hideUserPassword(user *types.User) *types.User {
	ret := *user
	ret.Password = ""
	return &ret
}
