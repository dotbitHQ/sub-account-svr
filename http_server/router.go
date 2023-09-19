package http_server

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal/static_files"
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
		v1.POST("/sub/account/mint/status", api_code.DoMonitorLog("mint_status"), cacheHandleShort, h.H.SubAccountMintStatus)
		v1.POST("/statistical/info", api_code.DoMonitorLog("statistical_info"), cacheHandleShort, h.H.StatisticalInfo)
		v1.POST("/distribution/list", api_code.DoMonitorLog("distribution_list"), cacheHandleShort, h.H.DistributionList)
		v1.POST("/currency/list", api_code.DoMonitorLog("currency_list"), cacheHandleShort, h.H.CurrencyList)
		v1.POST("/config/auto_mint/get", api_code.DoMonitorLog("config_auto_mint_get"), cacheHandleShort, h.H.ConfigAutoMintGet)
		v1.POST("/price/rule/list", api_code.DoMonitorLog("price_rule_list"), cacheHandleShort, h.H.PriceRuleList)
		v1.POST("/preserved/rule/list", api_code.DoMonitorLog("preserved_rule_list"), cacheHandleShort, h.H.PreservedRuleList)
		v1.POST("/auto/payment/list", api_code.DoMonitorLog("auto_payment_list"), cacheHandleShort, h.H.AutoPaymentList)
		v1.POST("/auto/order/info", api_code.DoMonitorLog("auto_order_info"), cacheHandleShort, h.H.AutoOrderInfo)
		v1.POST("/mint/config/get", api_code.DoMonitorLog("mint_config_get"), cacheHandleShort, h.H.MintConfigGet)
		v1.StaticFS("/static", http.FS(static_files.MintJs))

		v1.POST("/sub/account/init", api_code.DoMonitorLog("account_init"), h.H.SubAccountInit)               // enable_sub_account
		v1.POST("/sub/account/init/free", api_code.DoMonitorLog("account_init_free"), h.H.SubAccountInitFree) // enable_sub_account
		v1.POST("/sub/account/check", api_code.DoMonitorLog("account_check"), cacheHandleShort, h.H.SubAccountCheck)
		v1.POST("/sub/account/create", api_code.DoMonitorLog("account_create"), h.H.SubAccountCreateNew)            // create_sub_account
		v1.POST("/sub/account/renew", api_code.DoMonitorLog("account_renew"), h.H.SubAccountRenew)                  // renew_sub_account
		v1.POST("/sub/account/renew/check", api_code.DoMonitorLog("account_renew_check"), h.H.SubAccountRenewCheck) // renew_sub_account_check
		v1.POST("/sub/account/edit", api_code.DoMonitorLog("account_edit"), h.H.SubAccountEditNew)                  // edit_sub_account
		v1.POST("/owner/profit", api_code.DoMonitorLog("owner_profit"), h.H.OwnerProfit)
		v1.POST("/profit/withdraw", api_code.DoMonitorLog("profit_withdraw"), h.H.ProfitWithdraw)
		//v1.POST("/custom/script/set", api_code.DoMonitorLog("custom_script"), h.H.CustomScript)
		//v1.POST("/custom/script/info", api_code.DoMonitorLog("custom_script_info"), h.H.CustomScriptInfo)
		//v1.POST("/custom/script/price", api_code.DoMonitorLog("mint_price"), cacheHandleShort, h.H.CustomScriptPrice)
		v1.POST("/transaction/send", api_code.DoMonitorLog("tx_send"), h.H.TransactionSendNew)
		v1.POST("/mint/config/update", api_code.DoMonitorLog("mint_config_update"), h.H.MintConfigUpdate)
		v1.POST("/config/auto_mint/update", api_code.DoMonitorLog("config_auto_mint_update"), h.H.ConfigAutoMintUpdate)
		v1.POST("/price/rule/update", api_code.DoMonitorLog("price_rule_update"), h.H.PriceRuleUpdate)
		v1.POST("/preserved/rule/update", api_code.DoMonitorLog("preserved_rule_update"), h.H.PreservedRuleUpdate)
		v1.POST("/auto/account/search", api_code.DoMonitorLog("auto_acc_search"), h.H.AutoAccountSearch)
		v1.POST("/auto/order/create", api_code.DoMonitorLog("auto_order_create"), h.H.AutoOrderCreate)
		v1.POST("/auto/order/hash", api_code.DoMonitorLog("auto_order_hash"), h.H.AutoOrderHash)
		v1.POST("/currency/update", api_code.DoMonitorLog("currency_update"), h.H.CurrencyUpdate)
		//v1.POST("/mint/config/send", api_code.DoMonitorLog("mint_config_send"), h.H.MintConfigSend)
		v1.POST("/approval/enable", api_code.DoMonitorLog("approval_enable"), h.H.ApprovalEnable)
		v1.POST("/approval/delay", api_code.DoMonitorLog("approval_delay"), h.H.ApprovalDelay)
		v1.POST("/approval/revoke", api_code.DoMonitorLog("approval_revoke"), h.H.ApprovalRevoke)
		v1.POST("/approval/fulfill", api_code.DoMonitorLog("approval_fulfill"), h.H.ApprovalFulfill)
	}
	internalV1 := h.internalEngine.Group("v1")
	{
		internalV1.POST("/internal/smt/info", h.H.SmtInfo)
		internalV1.POST("/internal/smt/check", h.H.SmtCheck)
		internalV1.POST("/internal/smt/update", h.H.SmtUpdate)
		internalV1.POST("/internal/smt/syncTree", h.H.SmtSync)

		//internalV1.POST("/internal/sub/account/mint", h.H.InternalSubAccountMintNew)
		internalV1.POST("/owner/payment/export", h.H.OwnerPaymentExport)
		internalV1.POST("/unipay/notice", h.H.UniPayNotice)
		internalV1.POST("/service/provider/withdraw", h.H.ServiceProviderWithdraw)
		internalV1.POST("/service/provider/withdraw2", h.H.ServiceProviderWithdraw2)
		internalV1.POST("/internal/recycle/account", h.H.RecycleAccount)
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
