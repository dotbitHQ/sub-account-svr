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

type ReqCustomScriptPrice struct {
	SubAccount string `json:"sub_account"`
}

type RespCustomScriptPrice struct {
	CustomScriptPrice witness.CustomScriptPrice `json:"custom_script_price"`
}

func (h *HttpHandle) CustomScriptPrice(ctx *gin.Context) {
	var (
		funcName = "CustomScriptPrice"
		clientIp = GetClientIp(ctx)
		req      ReqCustomScriptPrice
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

	if err = h.doCustomScriptPrice(&req, &apiResp); err != nil {
		log.Error("doCustomScriptPrice err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCustomScriptPrice(req *ReqCustomScriptPrice, apiResp *api_code.ApiResp) error {
	var resp RespCustomScriptPrice

	index := strings.Index(req.SubAccount, ".")
	if index == -1 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sub-account is invalid")
		return nil
	}
	accountCharStr, err := h.DasCore.GetAccountCharSetList(req.SubAccount)
	//accountCharStr, err := common.AccountToAccountChars(req.SubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "AccountToAccountChars err")
		return nil
	}
	accLen := uint8(len(accountCharStr))

	parentAccount := req.SubAccount[index+1:]
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(parentAccount))
	log.Info("doCustomScriptPrice:", accLen, parentAccount, parentAccountId)

	// custom-script
	subAccLiveCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	detail := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
	if detail.HasCustomScriptArgs() {
		customScriptInfo, err := h.DbDao.GetCustomScriptInfo(parentAccountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return fmt.Errorf("GetCustomScriptInfo err: %s", err.Error())
		} else if customScriptInfo.Id > 0 {
			outpoint := common.String2OutPointStruct(customScriptInfo.Outpoint)

			log.Info("doCustomScriptPrice:", customScriptInfo.Outpoint)
			resTx, err := h.DasCore.Client().GetTransaction(h.Ctx, outpoint.TxHash)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
				return fmt.Errorf("GetTransaction err: %s", err.Error())
			}

			_, customScriptConfig, err := witness.ConvertCustomScriptConfigByTx(resTx.Transaction)
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
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeNotExistCustomScriptConfigPrice, "not exist price")
			return nil
		}
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeNotExistCustomScriptConfigPrice, "not exist price")
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
