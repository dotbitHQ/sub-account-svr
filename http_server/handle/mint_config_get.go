package handle

import (
	"das_sub_account/http_server/api_code"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqMintConfigGet struct {
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) MintConfigGet(ctx *gin.Context) {
	var (
		funcName               = "MintConfigGet"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqMintConfigGet
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

	if err = h.doMintConfigGet(&req, &apiResp); err != nil {
		log.Error("doMintConfigGet err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doMintConfigGet(req *ReqMintConfigGet, apiResp *api_code.ApiResp) error {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(accountId, apiResp); err != nil {
		return err
	}

	mintConfig, err := h.DbDao.GetMintConfig(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	apiResp.ApiRespOK(mintConfig)
	return nil
}
