package http_server

import (
	"context"
	"das_sub_account/http_server/handle"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LbHttpServer struct {
	Ctx     context.Context
	Address string
	H       *handle.LBHttpHandle
	engine  *gin.Engine
	srv     *http.Server
}

func (h *LbHttpServer) Run() {
	h.engine = gin.New()

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
}

func (h *LbHttpServer) Shutdown() {
	if h.srv != nil {
		log.Warn("http server Shutdown ... ")
		if err := h.srv.Shutdown(h.Ctx); err != nil {
			log.Error("http server Shutdown err:", err.Error())
		}
	}
}
