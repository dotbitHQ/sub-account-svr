package handle

import (
	"das_sub_account/http_server/api_code"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
)

type ReqOwnerProfit struct {
	core.ChainTypeAddress
	Account string `json:"account"`
}

type RespOwnerProfit struct {
	OwnerProfit string `json:"owner_profit"`
	BitProfit   string `json:"bit_profit"`
}

func (h *HttpHandle) OwnerProfit(ctx *gin.Context) {
	var (
		funcName = "OwnerProfit"
		clientIp = GetClientIp(ctx)
		req      ReqOwnerProfit
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

	if err = h.doOwnerProfit(&req, &apiResp); err != nil {
		log.Error("doOwnerProfit err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doOwnerProfit(req *ReqOwnerProfit, apiResp *api_code.ApiResp) error {
	var resp RespOwnerProfit

	hexAddress, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
		return nil
	}
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "account is nil")
		return nil
	}

	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "GetAccountInfoByAccountId err")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil
	} else if acc.OwnerChainType != hexAddress.ChainType || !strings.EqualFold(acc.Owner, hexAddress.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "owner permission required")
		return nil
	}

	subAccLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "GetSubAccountCell err")
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}

	detail := witness.ConvertSubAccountCellOutputData(subAccLiveCell.OutputData)
	log.Info("doOwnerProfit:", req.Account, detail.OwnerProfit, detail.DasProfit)

	decOwnerProfit, err := decimal.NewFromString(fmt.Sprintf("%d", detail.OwnerProfit))
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "format owner profit err")
		return fmt.Errorf("decimal.NewFromString err:%s", err.Error())
	}
	decOwnerProfit = decOwnerProfit.DivRound(decimal.NewFromInt(int64(common.OneCkb)), 8)
	resp.OwnerProfit = decOwnerProfit.String()

	decDasProfit, _ := decimal.NewFromString(fmt.Sprintf("%d", detail.DasProfit))
	decDasProfit = decDasProfit.DivRound(decimal.NewFromInt(int64(common.OneCkb)), 8)
	resp.BitProfit = decDasProfit.String()

	apiResp.ApiRespOK(resp)
	return nil
}
