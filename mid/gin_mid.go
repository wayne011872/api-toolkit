package mid

import (
	"github.com/gin-gonic/gin"
	"github.com/wayne011872/api-toolkit/errors"
)

type GinMiddle interface {
	errors.ApiErrorHandler
	Handler() gin.HandlerFunc
}

func NewGinMiddle(handler gin.HandlerFunc) GinMiddle {
	return &baseMiddle{
		handler: handler,
	}
}

type baseMiddle struct {
	handler gin.HandlerFunc
	errors.CommonApiErrorHandler
}

func (m *baseMiddle) Handler() gin.HandlerFunc {
	return m.handler
}
