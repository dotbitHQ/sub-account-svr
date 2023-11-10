package handle

import (
	"crypto/md5"
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
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

type ReqMintConfigUpdate struct {
	core.ChainTypeAddress
	Account         string        `json:"account" binding:"required"`
	Title           string        `json:"title" binding:"required"`
	Desc            string        `json:"desc" binding:"required"`
	Benefits        string        `json:"benefits" binding:"required"`
	Links           []tables.Link `json:"links"`
	BackgroundColor string        `json:"background_color"`
	Timestamp       int64         `json:"timestamp" binding:"required"`
	MintSuccessPage []struct {
		Type string `json:"type"`
		Url  string `json:"url"`
	} `json:"mint_success_page"`
}

type RespMintConfigUpdate struct {
	SignInfoList
}

const (
	ActionMintConfigUpdate string = "Update-Mint-Config"
)

func (r *ReqMintConfigUpdate) GetSignInfo() (signKey, signMsg, reqDataStr string) {
	reqData, _ := json.Marshal(r)
	reqDataStr = string(reqData)
	signKey = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s_%d", reqDataStr, time.Now().UnixNano()))))
	signMsg = common.DotBitPrefix + hex.EncodeToString(common.Blake2b(reqData))
	return
}

func (h *HttpHandle) MintConfigUpdate(ctx *gin.Context) {
	var (
		funcName               = "MintConfigUpdate"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqMintConfigUpdate
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

	if err = h.doMintConfigUpdate(&req, &apiResp); err != nil {
		log.Error("doMintConfigUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doMintConfigUpdate(req *ReqMintConfigUpdate, apiResp *api_code.ApiResp) error {
	var resp RespMintConfigUpdate
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

	action := ActionMintConfigUpdate
	if err := h.check(address, req.Account, action, apiResp); err != nil {
		return err
	}

	if time.UnixMilli(req.Timestamp).Add(time.Minute * 10).Before(time.Now()) {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params timestamp invalid")
		return nil
	}

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
	resp.SignList = []txbuilder.SignData{{
		SignType: signType,
		SignMsg:  signMsg,
	}}

	apiResp.ApiRespOK(resp)
	return nil
}
