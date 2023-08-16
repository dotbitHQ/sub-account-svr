package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqCustomScriptInfo struct {
	Account string `json:"account"`
}

type RespCustomScriptInfo struct {
	CustomScriptArgs   string                              `json:"custom_script_args"`
	CustomScriptConfig map[uint8]witness.CustomScriptPrice `json:"custom_script_config"`
}

func (h *HttpHandle) CustomScriptInfo(ctx *gin.Context) {
	var (
		funcName               = "CustomScriptInfo"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCustomScriptInfo
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doCustomScriptInfo(&req, &apiResp); err != nil {
		log.Error("doCustomScriptInfo err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCustomScriptInfo(req *ReqCustomScriptInfo, apiResp *api_code.ApiResp) error {
	var resp RespCustomScriptInfo
	resp.CustomScriptConfig = make(map[uint8]witness.CustomScriptPrice)

	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	// custom-script
	subAccLiveCell, err := h.DasCore.GetSubAccountCell(parentAccountId)
	if err != nil {
		doGetSubAccountCellErr(err, apiResp)
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	detail := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
	if detail.HasCustomScriptArgs() {
		resp.CustomScriptArgs = common.Bytes2Hex(detail.CustomScriptArgs)
		// config
		customScriptInfo, err := h.DbDao.GetCustomScriptInfo(parentAccountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return fmt.Errorf("GetCustomScriptInfo err: %s", err.Error())
		} else if customScriptInfo.Id > 0 {
			outpoint := common.String2OutPointStruct(customScriptInfo.Outpoint)

			log.Info("doCustomScriptInfo:", customScriptInfo.Outpoint)
			resTx, err := h.DasCore.Client().GetTransaction(h.Ctx, outpoint.TxHash)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
				return fmt.Errorf("GetTransaction err: %s", err.Error())
			}

			_, customScriptConfig, err := witness.ConvertCustomScriptConfigByTx(resTx.Transaction)
			if err != nil {
				if err != witness.ErrNotExistCustomScriptConfig {
					apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
					return fmt.Errorf("ConvertCustomScriptConfigByTx err: %s", err.Error())
				}
			} else {
				resp.CustomScriptConfig = customScriptConfig.Body
			}
		}
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func doGetSubAccountCellErr(err error, apiResp *api_code.ApiResp) {
	if err == nil || apiResp == nil {
		return
	}
	if strings.Contains(err.Error(), "sub account cell len") {
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, err.Error())
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
	}
}
