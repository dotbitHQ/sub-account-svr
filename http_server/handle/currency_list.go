package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

const (
	LabelSubDIDApp = "subdid.app"
)

type ReqCurrencyList struct {
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) CurrencyList(ctx *gin.Context) {
	var (
		funcName               = "CurrencyList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCurrencyList
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

	if err = h.doCurrencyList(&req, &apiResp); err != nil {
		log.Error("doCurrencyList err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCurrencyList(req *ReqCurrencyList, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(accountId, apiResp); err != nil {
		return err
	}

	paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	result := make([]tables.PaymentConfigElement, 0)
	for _, v := range config.Cfg.Das.AutoMint.SupportPaymentToken {
		token, err := h.DbDao.GetTokenById(v)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
			return err
		}
		cfg := tables.PaymentConfigElement{
			Enable:     false,
			TokenID:    v,
			Symbol:     token.Symbol,
			HaveRecord: false,
			Price:      token.Price,
			Decimals:   token.Decimals,
		}
		if userPaymentCfg, ok := paymentConfig.CfgMap[v]; ok && userPaymentCfg.Enable {
			cfg.Enable = true
		}

		if recordKeys, ok := common.TokenId2RecordKeyMap[v]; ok {
			record, err := h.DbDao.GetRecordsByAccountIdAndTypeAndLabel(accountId, "address", LabelSubDIDApp, recordKeys)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
				return err
			} else if record.Id > 0 {
				cfg.HaveRecord = true
			}
		}

		result = append(result, cfg)
	}
	apiResp.ApiRespOK(result)
	return nil
}
