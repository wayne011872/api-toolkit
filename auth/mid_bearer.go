package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wayne011872/api-toolkit/errors"
)

func NewGinBearAuthMid(isMatchHost bool) GinAuthMidInter {
	return &bearAuthMiddle{
		authMap:     make(map[string]uint8),
		groupMap:    make(map[string][]ApiPerm),
		isMatchHost: isMatchHost,
	}
}

func (lm *bearAuthMiddle) GetName() string {
	return "auth"
}

type bearAuthMiddle struct {
	errors.CommonApiErrorHandler
	authMap     map[string]uint8
	groupMap    map[string][]ApiPerm
	isMatchHost bool
}

type ctxKey string

const (
	authValue          = uint8(1 << iota)
	BearerAuthTokenKey = "Authorization"

	_KEY_USER_INFO     = "api_toolkit_user_info"
	_CTX_KEY_USER_INFO = ctxKey(_KEY_USER_INFO)
)

func SetReqUserToCtx(ctx context.Context, user ReqUser) context.Context {
	return context.WithValue(ctx, _CTX_KEY_USER_INFO, user)
}

func GetReqUserFromCtx(ctx context.Context) ReqUser {
	val := ctx.Value(_CTX_KEY_USER_INFO)
	if reqUser, ok := val.(ReqUser); ok {
		return reqUser
	}
	return nil
}

func SetReqUserToGin(c *gin.Context, user ReqUser) *gin.Context {
	c.Set(_KEY_USER_INFO, user)
	return c
}

func GetReqUserFromGin(c *gin.Context) ReqUser {
	data, ok := c.Get(_KEY_USER_INFO)
	if !ok {
		return nil
	}
	return data.(ReqUser)
}

func getPathKey(path, method string) string {
	return fmt.Sprintf("%s:%s", path, method)
}

func (am *bearAuthMiddle) AddAuthPath(path string, method string, isAuth bool, group []ApiPerm) {
	value := uint8(0)
	if isAuth {
		value = value | authValue
	}
	key := getPathKey(path, method)
	am.authMap[key] = uint8(value)
	am.groupMap[key] = group
}

func (am *bearAuthMiddle) IsAuth(path string, method string) bool {
	key := getPathKey(path, method)
	value, ok := am.authMap[key]
	if ok {
		return (value & authValue) > 0
	}
	return false
}

func (am *bearAuthMiddle) HasPerm(path, method string, perm []string) bool {
	key := fmt.Sprintf("%s:%s", path, method)
	groupAry, ok := am.groupMap[key]
	if !ok || groupAry == nil || len(groupAry) == 0 {
		return true
	}
	for _, g := range groupAry {
		if isStrInList(string(g), perm...) {
			return true
		}
	}
	return false
}

func (m *bearAuthMiddle) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method
		if path == "" {
			m.GinApiErrorHandler(c, errors.Error_Auth_Path_NotFound)
			c.Abort()
			return
		}
		if m.IsAuth(path, method) {
			authToken := c.GetHeader(BearerAuthTokenKey)
			if authToken == "" {
				m.GinApiErrorHandler(c, errors.Error_Auth_Miss_Token)
				c.Abort()
				return
			}

			if !strings.HasPrefix(authToken, "Bearer ") {
				m.GinApiErrorHandler(c, errors.Error_Auth_Invalid_Token)
				c.Abort()
				return
			}

			u, ok := c.Get(_KEY_USER_INFO)
			if !ok {
				m.GinApiErrorHandler(c, errors.Error_Auth_Miss_Token)
				c.Abort()
				return
			}
			reqUser := u.(ReqUser)

			host := getHost(c.Request)
			if m.isMatchHost && reqUser.GetHost() != host {
				m.GinApiErrorHandler(c, errors.Error_Auth_Host_Not_Match)
				c.Abort()
				return
			}

			if hasPerm := m.HasPerm(path, method, reqUser.GetPerms()); !hasPerm {
				m.GinApiErrorHandler(c, errors.Error_Auth_No_Perm)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func getHost(req *http.Request) string {
	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}
	return host
}

func isStrInList(input string, target ...string) bool {
	for _, paramName := range target {
		if input == paramName {
			return true
		}
	}
	return false
}
