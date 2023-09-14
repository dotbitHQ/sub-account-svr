package handle

import (
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqConfigAutoMintGet struct {
	Account string `json:"account" binding:"required"`
}

type RespConfigAutoMintGet struct {
	Enable bool `json:"enable"`
}

func (h *HttpHandle) ConfigAutoMintGet(ctx *gin.Context) {
	var (
		funcName               = "ConfigAutoMintGet"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqConfigAutoMintGet
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

	if err = h.doConfigAutoMintGet(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doConfigAutoMintGet(req *ReqConfigAutoMintGet, apiResp *api_code.ApiResp) error {
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	if err := h.checkForSearch(parentAccountId, apiResp); err != nil {
		return err
	}

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	subAccountCell, err := h.getSubAccountCell(baseInfo.ContractSubAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.OutPoint.Index])

	resp := RespConfigAutoMintGet{
		Enable: false,
	}
	if subAccountCellDetail.Flag == witness.FlagTypeCustomRule &&
		subAccountCellDetail.AutoDistribution == witness.AutoDistributionEnable {
		resp.Enable = true
	}
	apiResp.ApiRespOK(resp)
	return nil
}
