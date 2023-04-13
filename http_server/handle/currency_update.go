package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqCurrencyUpdate struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
	TokenID string `json:"token_id" binding:"required"`
	Enable  bool   `json:"enable"`
}

func (h *HttpHandle) CurrencyUpdate(ctx *gin.Context) {
	var (
		funcName               = "CurrencyUpdate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCurrencyUpdate
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doCurrencyUpdate(&req, &apiResp); err != nil {
		log.Error("doCurrencyUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCurrencyUpdate(req *ReqCurrencyUpdate, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}

	find := false
	for _, v := range config.Cfg.Das.SupportPaymentToken {
		if v == req.TokenID {
			find = true
			break
		}
	}
	if !find {
		err := fmt.Errorf("token_id: %s, no support now", req.TokenID)
		apiResp.ApiRespErr(api_code.ApiCodeNoSupportPaymentToken, err.Error())
		return err
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	paymentConfig.CfgMap[req.TokenID] = tables.PaymentConfigElement{
		Enable: req.Enable,
	}
	if err := h.DbDao.UpdatePaymentConfig(req.Account, &paymentConfig); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	return nil
}
