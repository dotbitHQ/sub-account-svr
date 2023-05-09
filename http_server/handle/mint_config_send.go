package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqMintConfigSend struct {
	core.ChainTypeAddress
	SignInfoList
}

func (h *HttpHandle) MintConfigSend(ctx *gin.Context) {
	var (
		funcName               = "MintConfigSend"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqMintConfigSend
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

	if err = h.doMintConfigSend(&req, &apiResp); err != nil {
		log.Error("doMintConfigSend err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doMintConfigSend(req *ReqMintConfigSend, apiResp *api_code.ApiResp) error {
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

	signMsg, err := h.RC.Red.Get(req.SignKey).Result()
	if err != nil && err != redis.Nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}
	if err == redis.Nil {
		apiResp.ApiRespErr(api_code.ApiCodeSignError, "sign expired")
		return errors.New("sign expired")
	}

	signData := &ReqMintConfigUpdate{}
	if err := json.Unmarshal([]byte(signMsg), signData); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "data error")
		return errors.New("data error")
	}

	if req.KeyInfo.Key != signData.ChainTypeAddress.KeyInfo.Key {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "no operation permission")
		return errors.New("no operation permission")
	}
	signMsg = common.DotBitPrefix + hex.EncodeToString(common.Blake2b([]byte(signMsg)))

	if _, err = doSignCheck(txbuilder.SignData{
		SignType: res.DasAlgorithmId,
		SignMsg:  signMsg,
	}, req.List[0].SignList[0].SignMsg, res.AddressHex, apiResp); err != nil {
		return fmt.Errorf("doSignCheck err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	h.RC.Red.Del(req.SignKey)

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(signData.Account))
	if err := h.DbDao.CreateUserConfigWithMintConfig(tables.UserConfig{
		Account:   signData.Account,
		AccountId: accountId,
	}, tables.MintConfig{
		Title:           signData.Title,
		Desc:            signData.Desc,
		Benefits:        signData.Benefits,
		Links:           signData.Links,
		BackgroundColor: signData.BackgroundColor,
	}); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to update mint config")
		return fmt.Errorf("CreateUserConfigWithMintConfig err: %s", err.Error())
	}

	apiResp.ApiRespOK(nil)
	return nil
}
