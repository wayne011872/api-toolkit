package apitool

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wayne011872/api-toolkit/auth"
	"github.com/wayne011872/api-toolkit/errors"
	"github.com/wayne011872/api-toolkit/mid"
)

type GinApiHandler struct {
	Handler func(c *gin.Context)
	Method  string
	Path    string
	Auth    bool
	Group   []auth.ApiPerm
}

type GinAPI interface {
	errors.ApiErrorHandler
	GetAPIs() []*GinApiHandler
}

type GinApiServer interface {
	AddAPIs(handlers ...GinAPI) GinApiServer
	Middles(mids ...mid.GinMiddle) GinApiServer
	SetServerErrorHandler(errors.GinServerErrorHandler) GinApiServer
	SetAuth(authmid auth.GinAuthMidInter) GinApiServer
	SetTrustedProxies([]string) GinApiServer
	SetPromhttp(c ...prometheus.Collector) GinApiServer
	Static(relativePath, root string) GinApiServer
	Run(port int) error
	errorHandler(c *gin.Context, err error)
	GetServer(port int) *http.Server
}

type ginApiServ struct {
	*gin.Engine
	service      string
	authMid      auth.GinAuthMidInter
	myErrHandler errors.GinServerErrorHandler
	apiMids      []gin.HandlerFunc
}

func (serv *ginApiServ) SetServerErrorHandler(handler errors.GinServerErrorHandler) GinApiServer {
	serv.myErrHandler = handler
	return serv
}

func (serv *ginApiServ) errorHandler(c *gin.Context, err error) {
	serv.myErrHandler(c, serv.service, err)
}

func (serv *ginApiServ) Static(relativePath, root string) GinApiServer {
	serv.Engine.Static(relativePath, root)
	return serv
}

func (serv *ginApiServ) SetAuth(authMid auth.GinAuthMidInter) GinApiServer {
	serv.authMid = authMid
	return serv
}

func (serv *ginApiServ) Middles(mids ...mid.GinMiddle) GinApiServer {
	for _, m := range mids {
		m.SetApiErrorHandler(serv.errorHandler)
		serv.apiMids = append(serv.apiMids, m.Handler())
		//serv.Engine.Use(m.Handler())
	}
	return serv
}

func (serv *ginApiServ) AddAPIs(apis ...GinAPI) GinApiServer {
	for _, api := range apis {
		api.SetApiErrorHandler(serv.errorHandler)
		for _, h := range api.GetAPIs() {
			if serv.authMid != nil {
				serv.authMid.AddAuthPath(h.Path, h.Method, h.Auth, h.Group)
			}
			switch h.Method {
			case "GET":
				serv.Engine.GET(h.Path, append(serv.apiMids, h.Handler)...)
			case "POST":
				serv.Engine.POST(h.Path, append(serv.apiMids, h.Handler)...)
			case "PUT":
				serv.Engine.PUT(h.Path, append(serv.apiMids, h.Handler)...)
			case "DELETE":
				serv.Engine.DELETE(h.Path, append(serv.apiMids, h.Handler)...)
			}
		}
	}
	return serv
}

func (serv *ginApiServ) SetTrustedProxies(proxies []string) GinApiServer {
	if len(proxies) == 0 {
		return serv
	}
	serv.Engine.ForwardedByClientIP = true
	serv.Engine.SetTrustedProxies(proxies)
	return serv
}

func (serv *ginApiServ) Run(port int) error {
	return serv.Engine.Run(":" + strconv.Itoa(port))
}

func (serv *ginApiServ) GetServer(port int) *http.Server {
	return &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: serv.Engine,
	}
}

func NewGinApiServer(mode string, service string) GinApiServer {
	gin.SetMode(mode)
	return &ginApiServ{
		Engine:  gin.New(),
		service: service,
	}
}

func (serv *ginApiServ) SetPromhttp(c ...prometheus.Collector) GinApiServer {
	prometheus.MustRegister(c...)
	serv.Engine.GET("/metrics", promGinHandler).Use()
	return serv
}
func promGinHandler(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}
