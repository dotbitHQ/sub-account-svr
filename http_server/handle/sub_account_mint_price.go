package handle

import (
	"das_sub_account/http_server/api_code"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqSubAccountMintPrice struct {
	SubAccount string `json:"sub_account"`
}

type RespSubAccountMintPrice struct {
	CustomScriptPrice witness.CustomScriptPrice `json:"custom_script_price"`
}

func (h *HttpHandle) SubAccountMintPrice(ctx *gin.Context) {
	var (
		funcName = "SubAccountMintPrice"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountMintPrice
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

	if err = h.doSubAccountMintPrice(&req, &apiResp); err != nil {
		log.Error("doSubAccountMintPrice err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountMintPrice(req *ReqSubAccountMintPrice, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountMintPrice

	index := strings.Index(req.SubAccount, ".")
	if index == -1 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sub-account is invalid")
		return nil
	}
	accLen := common.GetAccountLength(req.SubAccount[:index])
	parentAccount := req.SubAccount[index:]
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(parentAccount))
	log.Info("doSubAccountMintPrice:", accLen, parentAccount, parentAccountId)

	customScriptInfo, err := h.DbDao.GetCustomScriptInfo(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return fmt.Errorf("GetCustomScriptInfo err: %s", err.Error())
	}
	outpoint := common.String2OutPointStruct(customScriptInfo.Outpoint)

	log.Info("doSubAccountMintPrice:", customScriptInfo.Outpoint)
	resTx, err := h.DasCore.Client().GetTransaction(h.Ctx, outpoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetTransaction err: %s", err.Error())
	}

	customScriptConfig, err := witness.ConvertCustomScriptConfigByTx(resTx.Transaction)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConvertCustomScriptConfigByTx err: %s", err.Error())
	}
	if accLen > customScriptConfig.MaxLength {
		accLen = customScriptConfig.MaxLength
	}
	if item, ok := customScriptConfig.Body[accLen]; ok {
		resp.CustomScriptPrice = item
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeNotExistCustomScriptConfigPrice, "not exist price")
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
