package http_server

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

func (h *HttpServer) initRouter() {
	shortExpireTime, shortDataTime, lockTime := time.Second*5, time.Minute*3, time.Minute
	cacheHandleShort := toolib.MiddlewareCacheByRedis(h.H.RC.Red, false, shortDataTime, lockTime, shortExpireTime, respHandle)

	log.Info("initRouter:", len(config.Cfg.Origins))
	if len(config.Cfg.Origins) > 0 {
		toolib.AllowOriginList = append(toolib.AllowOriginList, config.Cfg.Origins...)
	}
	h.internalEngine.Use(toolib.MiddlewareCors())
	h.engine.Use(toolib.MiddlewareCors())

	v1 := h.engine.Group("v1")
	{
		v1.POST("/version", cacheHandleShort, h.H.Version)
		v1.POST("/config/info", api_code.DoMonitorLog("config"), cacheHandleShort, h.H.ConfigInfo)
		v1.POST("/account/list", api_code.DoMonitorLog("account_list"), cacheHandleShort, h.H.AccountList)
		v1.POST("/account/detail", api_code.DoMonitorLog("account_detail"), cacheHandleShort, h.H.AccountDetail)
		v1.POST("/sub/account/list", api_code.DoMonitorLog("sub_account_list"), cacheHandleShort, h.H.SubAccountList)
		v1.POST("/transaction/status", api_code.DoMonitorLog("tx_status"), cacheHandleShort, h.H.TransactionStatus)
		v1.POST("/task/status", api_code.DoMonitorLog("task_status"), cacheHandleShort, h.H.TaskInfo)
		v1.POST("/sub/account/mint/status", api_code.DoMonitorLog("mint_status"), cacheHandleShort, h.H.SubAccountMintStatus)

		v1.POST("/sub/account/init", api_code.DoMonitorLog("account_init"), h.H.SubAccountInit) // enable_sub_account
		v1.POST("/sub/account/check", api_code.DoMonitorLog("account_check"), cacheHandleShort, h.H.SubAccountCheck)
		if config.Cfg.Server.RunMode == "normal" {
			v1.POST("/sub/account/create", api_code.DoMonitorLog("account_create"), h.H.SubAccountCreateNew) // create_sub_account
			v1.POST("/sub/account/edit", api_code.DoMonitorLog("account_edit"), h.H.SubAccountEditNew)       // edit_sub_account
		}
		v1.POST("/owner/profit", api_code.DoMonitorLog("owner_profit"), h.H.OwnerProfit)
		v1.POST("/profit/withdraw", api_code.DoMonitorLog("profit_withdraw"), h.H.ProfitWithdraw)
		v1.POST("/custom/script/set", api_code.DoMonitorLog("custom_script"), h.H.CustomScript)
		v1.POST("/custom/script/info", api_code.DoMonitorLog("custom_script_info"), h.H.CustomScriptInfo)
		v1.POST("/custom/script/price", api_code.DoMonitorLog("mint_price"), cacheHandleShort, h.H.CustomScriptPrice)
		v1.POST("/transaction/send", api_code.DoMonitorLog("tx_send"), h.H.TransactionSend) //

		// recycle_expired_account_by_keeper, del smt info
		// renew_sub_account( need to renew by self(single) or renew by register server(multi) )
		// recycle_sub_account(0,0->taskId,taskIndex)
	}
	internalV1 := h.internalEngine.Group("v1")
	{
		internalV1.POST("/internal/smt/info", h.H.SmtInfo)
		internalV1.POST("/internal/smt/check", h.H.SmtCheck)
		internalV1.POST("/internal/smt/update", h.H.SmtUpdate)

		//internalV1.POST("/internal/sub/account/create", h.H.InternalSubAccountCreate)
		internalV1.POST("/internal/sub/account/mint", h.H.InternalSubAccountMint)
	}
}

func respHandle(c *gin.Context, res string, err error) {
	if err != nil {
		log.Error("respHandle err:", err.Error())
		c.AbortWithStatusJSON(http.StatusOK, api_code.ApiRespErr(http.StatusInternalServerError, err.Error()))
	} else if res != "" {
		var respMap map[string]interface{}
		_ = json.Unmarshal([]byte(res), &respMap)
		c.AbortWithStatusJSON(http.StatusOK, respMap)
	}
}
