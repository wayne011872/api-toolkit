package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wayne011872/api-toolkit/errors"
)

func NewMockAuthMid() GinAuthMidInter {
	return &mockAuthMiddle{}
}

type mockAuthMiddle struct {
	errors.CommonApiErrorHandler
}

func (am *mockAuthMiddle) AddAuthPath(path string, method string, isAuth bool, group []ApiPerm) {
}

const (
	_MOCK_HEADER_KEY_UID     = "Mock_User_UID"
	_MOCK_HEADER_KEY_ACCOUNT = "Mock_User_ACC"
	_MOCK_HEADER_KEY_NAME    = "Mock_User_NAM"
	_MOCK_HEADER_KEY_ROLES   = "Mock_User_Roles"
)

func (am *mockAuthMiddle) IsAuth(path string, method string) bool {
	return true
}
func (am *mockAuthMiddle) HasPerm(path, method string, perm []string) bool {
	return true
}

func (am *mockAuthMiddle) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader(_MOCK_HEADER_KEY_UID)
		if userID == "" {
			userID = "mock-id"
		}
		userAcc := c.GetHeader(_MOCK_HEADER_KEY_ACCOUNT)
		if userAcc == "" {
			userAcc = "mock-account"
		}
		userName := c.GetHeader(_MOCK_HEADER_KEY_NAME)
		if userName == "" {
			userName = "mock-name"
		}
		roles := strings.Split(c.GetHeader(_MOCK_HEADER_KEY_ROLES), ",")
		if len(roles) == 0 {
			roles = []string{"mock"}
		}
		c.Set(
			_KEY_USER_INFO,
			NewReqUser(getHost(c.Request), userID, userAcc, userName, roles, "access"),
		)
		c.Next()
	}
}

func NewReqUser(host string, uid string, account string, name string, roles []string, usage string) ReqUser {
	return &reqUserImpl{
		host:    host,
		uid:     uid,
		account: account,
		name:    name,
		roles:   roles,
		usage:   usage,
	}
}
