package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqMintConfigUpdate struct {
	core.ChainTypeAddress
	Account  string        `json:"account" binding:"required"`
	Title    string        `json:"title" binding:"required"`
	Desc     string        `json:"desc" binding:"required"`
	Benefits string        `json:"benefits" binding:"required"`
	Links    []tables.Link `json:"links"`
}

func (h *HttpHandle) MintConfigUpdate(ctx *gin.Context) {
	var (
		funcName = "MintConfigUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqMintConfigUpdate
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

	if err = h.doMintConfigUpdate(&req, &apiResp); err != nil {
		log.Error("doSubAccountList err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doMintConfigUpdate(req *ReqMintConfigUpdate, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)
	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}

	if err := h.DbDao.UpdateMintConfig(req.Account, tables.MintConfig{
		Title:    req.Title,
		Desc:     req.Desc,
		Benefits: req.Benefits,
		Links:    req.Links,
	}); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "db error")
		return err
	}
	return nil
}
