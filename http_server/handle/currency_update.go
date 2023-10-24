package handle

import (
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/consts"
	"das_sub_account/internal"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
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

const ()

func (r *ReqCurrencyUpdate) GetSignInfo() (signKey, signMsg, reqDataStr string) {
	reqData, _ := json.Marshal(r)
	reqDataStr = string(reqData)
	signKey = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s_%d", reqDataStr, time.Now().UnixNano()))))
	signMsg = common.DotBitPrefix + hex.EncodeToString(common.Blake2b(reqData))
	return
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
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doCurrencyUpdate(&req, &apiResp); err != nil {
		log.Error("doCurrencyUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
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

	action := consts.ActionCurrencyUpdate
	if err := h.check(address, req.Account, action, apiResp); err != nil {
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

	// check price
	//accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	//if req.TokenId == string(tables.TokenIdStripeUSD) {
	//	ruleConfig, err := h.DbDao.GetRuleConfigByAccountId(accountId)
	//	if err != nil {
	//		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to search rule config")
	//		return fmt.Errorf("GetRuleConfigByAccountId err: %s", err.Error())
	//	}
	//	ruleTx, err := h.DasCore.Client().GetTransaction(h.Ctx, types.HexToHash(ruleConfig.TxHash))
	//	if err != nil {
	//		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rule tx")
	//		return fmt.Errorf("GetTransaction err: %s", err.Error())
	//	}
	//	rulePrice := witness.NewSubAccountRuleEntity(req.Account)
	//	if err = rulePrice.ParseFromTx(ruleTx.Transaction, common.ActionDataTypeSubAccountPriceRules); err != nil {
	//		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to search rules")
	//		return fmt.Errorf("ParseFromTx err: %s", err.Error())
	//	}
	//	for _, v := range rulePrice.Rules {
	//		if v.Price < 0.52 {
	//			apiResp.ApiRespErr(http_api.ApiCodeAmountIsTooLow, "Prices must not be lower than 0.52$")
	//			return nil
	//		}
	//	}
	//}
	//
	signKey, signMsg, reqDataStr := req.GetSignInfo()
	if err := h.RC.Red.Set(signKey, reqDataStr, time.Minute*10).Err(); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}

	// cache
	if err = h.RC.SetSignTxCache(signKey, reqDataStr); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeCacheError, "cache err")
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	//
	signType := res.DasAlgorithmId
	if signType == common.DasAlgorithmIdEth712 {
		signType = common.DasAlgorithmIdEth
	}
	resp.Action = action
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: []txbuilder.SignData{{
			SignType: signType,
			SignMsg:  signMsg,
		}},
	})

	apiResp.ApiRespOK(resp)
	return nil
}
