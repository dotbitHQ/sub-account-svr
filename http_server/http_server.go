package http_server

import (
	"context"
	"das_sub_account/http_server/handle"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	log = logger.NewLogger("http_server", logger.LevelDebug)
)

type HttpServer struct {
	Ctx             context.Context
	Address         string
	InternalAddress string
	H               *handle.HttpHandle
	engine          *gin.Engine
	internalEngine  *gin.Engine
	srv             *http.Server
	internalSrv     *http.Server
}

func (h *HttpServer) Run() {
	h.engine = gin.New()
	h.internalEngine = gin.New()

	h.initRouter()

	h.srv = &http.Server{
		Addr:    h.Address,
		Handler: h.engine,
	}
	go func() {
		if err := h.srv.ListenAndServe(); err != nil {
			log.Error("http_server run err:", err)
		}
	}()

	h.internalSrv = &http.Server{
		Addr:    h.InternalAddress,
		Handler: h.internalEngine,
	}
	go func() {
		if err := h.internalSrv.ListenAndServe(); err != nil {
			log.Error("http_server internal run err:", err)
		}
	}()
}

func (h *HttpServer) Shutdown() {
	if h.srv != nil {
		log.Warn("http server Shutdown ... ")
		if err := h.srv.Shutdown(h.Ctx); err != nil {
			log.Error("http server Shutdown err:", err.Error())
		}
	}
	if h.internalSrv != nil {
		log.Warn("http server internal Shutdown ... ")
		if err := h.internalSrv.Shutdown(h.Ctx); err != nil {
			log.Error("http server internal Shutdown err:", err.Error())
		}
	}
}
