package http_server

import (
	"das_sub_account/config"
	"github.com/scorpiotzh/toolib"
)

func (h *LbHttpServer) initRouter() {
	log.Info("initRouter:", len(config.Cfg.Origins))
	if len(config.Cfg.Origins) > 0 {
		toolib.AllowOriginList = append(toolib.AllowOriginList, config.Cfg.Origins...)
	}
	h.engine.Use(toolib.MiddlewareCors())
	v1 := h.engine.Group("v1")
	{
		v1.POST("/version", h.H.LBProxy)
		v1.POST("/config/info", h.H.LBProxy)
		v1.POST("/account/list", h.H.LBProxy)
		v1.POST("/account/detail", h.H.LBProxy)
		v1.POST("/sub/account/list", h.H.LBProxy)
		v1.POST("/transaction/status", h.H.LBProxy)
		v1.POST("/task/status", h.H.LBProxy)
		v1.POST("/sub/account/mint/status", h.H.LBProxy)
		v1.POST("/custom/script/info", h.H.LBProxy)
		v1.POST("/custom/script/price", h.H.LBProxy)
		v1.POST("/owner/profit", h.H.LBProxy)
		v1.POST("/sub/account/check", h.H.LBProxy)

		v1.POST("/sub/account/init", h.H.LBProxy) // enable_sub_account
		v1.POST("/custom/script/set", h.H.LBProxy)
		v1.POST("/profit/withdraw", h.H.LBProxy)
		v1.POST("/sub/account/edit", h.H.LBProxy)

		v1.POST("/sub/account/create", h.H.LBSubAccountCreate) // create_sub_account
		v1.POST("/transaction/send", h.H.LBTransactionSend)

		v1.POST("/new/sub/account/create", h.H.LBProxy)
		v1.POST("/new/sub/account/edit", h.H.LBProxy)
		v1.POST("/new/transaction/send", h.H.LBTransactionSend)
	}
}
