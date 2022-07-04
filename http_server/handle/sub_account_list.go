package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqSubAccountList struct {
	Pagination
	Account string `json:"account"`
	api_code.ChainTypeAddress
	chainType common.ChainType
	address   string
	Keyword   string          `json:"keyword"`
	Category  tables.Category `json:"category"`
}

type RespSubAccountList struct {
	Total int64         `json:"total"`
	List  []AccountData `json:"list"`
}

func (h *HttpHandle) SubAccountList(ctx *gin.Context) {
	var (
		funcName = "SubAccountList"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountList
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

	if err = h.doSubAccountList(&req, &apiResp); err != nil {
		log.Error("doSubAccountList err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountList(req *ReqSubAccountList, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountList
	resp.List = make([]AccountData, 0)

	// check params
	if req.ChainTypeAddress.KeyInfo.Key != "" {
		addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
			return nil
		}
		req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex
	}

	// check params
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid")
		return nil
	}

	// get sub account list
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	list, err := h.DbDao.GetSubAccountListByParentAccountId(accountId, req.chainType, req.address, req.Keyword, req.GetLimit(), req.GetOffset(), req.Category)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query sub account list")
		return fmt.Errorf("GetSubAccountListByParentAccountId err: %s", err.Error())
	}
	for _, v := range list {
		tmp := h.accountInfoToAccountData(v)
		resp.List = append(resp.List, tmp)
	}

	// total
	count, err := h.DbDao.GetSubAccountListTotalByParentAccountId(accountId, req.chainType, req.address, req.Keyword, req.Category)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query sub account total")
		return fmt.Errorf("GetSubAccountListTotalByParentAccountId err: %s", err.Error())
	}
	resp.Total = count

	apiResp.ApiRespOK(resp)
	return nil
}
