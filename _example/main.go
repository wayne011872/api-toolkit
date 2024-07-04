package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apitool "github.com/wayne011872/api-toolkit"
	"github.com/wayne011872/api-toolkit/errors"
)

func main() {
	serverErrorHandler := func(c *gin.Context, service string, err error) {
		if err == nil {
			return
		}
		if apiErr, ok := err.(errors.ApiError); ok {
			c.AbortWithStatusJSON(apiErr.GetStatus(),
				map[string]interface{}{
					"status":   apiErr.GetStatus(),
					"error":    apiErr.Error(),
					"service":  service,
					"errorKey": apiErr.GetKey(),
				})
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				map[string]interface{}{
					"status":   http.StatusInternalServerError,
					"title":    err.Error(),
					"service":  service,
					"errorKey": "",
				})
		}
	}
	apitool.NewGinApiServer("debug", "test").
		SetServerErrorHandler(serverErrorHandler).
		AddAPIs(
			&testApiService{},
		).Run(8080)
}

type testApiService struct {
	errors.CommonApiErrorHandler
}

func (a *testApiService) GetAPIs() []*apitool.GinApiHandler {
	return []*apitool.GinApiHandler{
		{Path: "/v1/test", Handler: a.testHandler, Method: "GET", Auth: false},
	}
}

func (a *testApiService) testHandler(c *gin.Context) {
	a.GinApiErrorHandler(c, errors.New(http.StatusBadGateway, "test error"))
}
