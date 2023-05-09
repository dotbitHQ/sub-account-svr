package handle

import (
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"encoding/hex"
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
}

type RespCurrencyUpdate struct {
	SignInfoList
}

const (
	ActionCurrencyUpdate string = "Update-Currency"
)

func (r *ReqCurrencyUpdate) SignKey() string {
	key := fmt.Sprintf("%s%s%s%t%d", r.ChainTypeAddress.KeyInfo.Key, r.Account, r.TokenId, r.Enable, time.Now().UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

func (r *ReqCurrencyUpdate) SignMsg() string {
	sigMsg := ""
	if r.Enable {
		sigMsg = fmt.Sprintf("Enable %s on %d", r.TokenId, r.Timestamp)
	} else {
		sigMsg = fmt.Sprintf("Disable %s on %d", r.TokenId, r.Timestamp)
	}
	log.Info("SigMsg2:", sigMsg)
	return common.DotBitPrefix + hex.EncodeToString(common.Blake2b([]byte(sigMsg)))
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
	var resp RespCurrencyUpdate
	resp.List = make([]SignInfo, 0)

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

	if time.UnixMilli(req.Timestamp).Add(time.Minute * 10).Before(time.Now()) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params timestamp invalid")
		return nil
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

	// cache
	signKey := req.SignKey()
	cacheStr := toolib.JsonString(req)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "signature expired")
		return fmt.Errorf("signature expired")
	}

	//
	signType := res.DasAlgorithmId
	if signType == common.DasAlgorithmIdEth712 {
		signType = common.DasAlgorithmIdEth
	}
	resp.Action = ActionCurrencyUpdate
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: []txbuilder.SignData{{
			SignType: signType,
			SignMsg:  req.SignMsg(),
		}},
	})

	apiResp.ApiRespOK(resp)

	return nil
}
