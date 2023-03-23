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
	"strings"
)

const (
	LabelSubDIDApp = "subdid.app"
)

type ReqCurrencyList struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) CurrencyList(ctx *gin.Context) {
	var (
		funcName = "CurrencyList"
		clientIp = GetClientIp(ctx)
		req      ReqCurrencyList
		apiResp  api_code.ApiResp
		err      error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req))

	if err = h.doCurrencyList(&req, &apiResp); err != nil {
		log.Error("doCurrencyList err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCurrencyList(req *ReqCurrencyList, apiResp *api_code.ApiResp) error {
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
	address := res.AddressHex
	if strings.HasPrefix(res.AddressHex, common.HexPreFix) {
		address = strings.ToLower(res.AddressHex)
	}
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	result := make([]tables.PaymentConfigElement, 0)
	for _, v := range config.Cfg.Das.SupportPaymentToken {
		splitToken := map[string]bool{}
		for _, v := range strings.Split(v, "_") {
			splitToken[v] = true
		}

		cfg := tables.PaymentConfigElement{
			TokenID: v,
		}
		if userPaymentCfg, ok := paymentConfig.CfgMap[v]; ok && userPaymentCfg.Enable {
			cfg.Enable = true
			records, err := h.DbDao.GetRecordsByAccountIdAndLabel(accountId, LabelSubDIDApp)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
				return err
			}
			for _, record := range records {
				if splitToken[record.Key] {
					cfg.HaveRecord = true
					break
				}
			}
		}
		result = append(result, cfg)
	}
	apiResp.ApiRespOK(result)
	return nil
}
