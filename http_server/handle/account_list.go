package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqAccountList struct {
	Pagination
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
	Category  tables.Category `json:"category"`
	Keyword   string          `json:"keyword"`
}

type RespAccountList struct {
	Total int64         `json:"total"`
	List  []AccountData `json:"list"`
}

func (h *HttpHandle) AccountList(ctx *gin.Context) {
	var (
		funcName               = "AccountList"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqAccountList
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

	if err = h.doAccountList(&req, &apiResp); err != nil {
		log.Error("doAccountList err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doAccountList(req *ReqAccountList, apiResp *api_code.ApiResp) error {
	var resp RespAccountList
	resp.List = make([]AccountData, 0)

	// check params
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex

	// account list
	list, err := h.DbDao.GetAccountList(req.chainType, req.address, req.GetLimit(), req.GetOffset(), req.Category, req.Keyword)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account list")
		return fmt.Errorf("GetAccountList err: %s", err.Error())
	}
	for _, v := range list {
		tmp := h.accountInfoToAccountData(v)
		resp.List = append(resp.List, tmp)
	}

	// total
	count, err := h.DbDao.GetAccountListTotal(req.chainType, req.address, req.Category, req.Keyword)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account total")
		return fmt.Errorf("GetAccountListTotal err: %s", err.Error())
	}
	resp.Total = count

	apiResp.ApiRespOK(resp)
	return nil
}
