package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type ReqCurrencyUpdate struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	TokenId   string `json:"token_id" binding:"required"`
	Enable    bool   `json:"enable"`
	Timestamp int64  `json:"timestamp" binding:"required"`
	Signature string `json:"signature" binding:"required"`
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
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)
	if err := h.check(address, req.Account, apiResp); err != nil {
		return err
	}

	token, err := h.DbDao.GetTokenById(req.TokenId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}

	var signMsg string
	if req.Enable {
		signMsg = fmt.Sprintf("Enable %s on %d", token.Symbol, req.Timestamp)
	} else {
		signMsg = fmt.Sprintf("Disable %s on %d", token.Symbol, req.Timestamp)
	}

	if _, err = doSignCheck(txbuilder.SignData{
		SignType: res.DasAlgorithmId,
		SignMsg:  signMsg,
	}, req.Signature, res.AddressHex, apiResp); err != nil {
		return fmt.Errorf("doSignCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	if time.UnixMilli(req.Timestamp).Add(time.Minute * 10).Before(time.Now()) {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "signature expired")
		return fmt.Errorf("timestamp invalid")
	}

	find := false
	for _, v := range config.Cfg.Das.AutoMint.SupportPaymentToken {
		if v == req.TokenId {
			find = true
			break
		}
	}
	if !find {
		err := fmt.Errorf("token_id: %s, no support now", req.TokenId)
		apiResp.ApiRespErr(api_code.ApiCodeNoSupportPaymentToken, err.Error())
		return err
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	paymentConfig, err := h.DbDao.GetUserPaymentConfig(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	paymentConfig.CfgMap[req.TokenId] = tables.PaymentConfigElement{
		Enable: req.Enable,
	}
	if err := h.DbDao.CreateUserConfigWithPaymentConfig(tables.UserConfig{
		Account:   req.Account,
		AccountId: accountId,
	}, paymentConfig); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to update payment config")
		return fmt.Errorf("CreateUserConfigWithMintConfig err: %s", err.Error())
	}
	return nil
}
