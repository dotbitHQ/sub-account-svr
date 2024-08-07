package handle

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/internal"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

func (h *HttpHandle) SubAccountInitFree(ctx *gin.Context) {
	var (
		funcName               = "SubAccountInitFree"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountInit
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doSubAccountInitFree(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doSubAccountInit err:", err.Error(), funcName, clientIp, ctx.Request.Context())
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountInitFree(ctx context.Context, req *ReqSubAccountInit, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountInit
	resp.List = make([]SignInfo, 0)
	resp.SignList = make([]txbuilder.SignData, 0)
	// check params
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex

	// check update
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	// check account
	acc, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionEnableSubAccount)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	if acc.OwnerChainType != req.chainType || !strings.EqualFold(acc.Owner, req.address) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "not have owner permission")
		return nil
	}

	// config cell
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount, common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}

	subAccountBasicCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	subAccountPreparedFeeCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	subAccountCommonFee, _ := molecule.Bytes2GoU64(builder.ConfigCellAccount.CommonFee().RawData())

	capacityNeed, capacityForChange := subAccountBasicCapacity+subAccountPreparedFeeCapacity+subAccountCommonFee, common.DasLockWithBalanceTypeMinCkbCapacity

	change, liveCells, err := h.getSvrBalance(ctx, paramBalance{
		svrLock:           h.ServerScript,
		capacityForNeed:   capacityNeed,
		capacityForChange: capacityForChange,
	})
	if err != nil {
		return doDasBalanceError(err, apiResp)
	}

	// build tx
	buildParams := paramsSubAccountInitTx{
		req:                req,
		acc:                acc,
		liveCells:          liveCells,
		subAccountCapacity: subAccountBasicCapacity + subAccountPreparedFeeCapacity,
		txFee:              subAccountCommonFee,
		change:             change,
		feeDasLock:         h.ServerScript,
	}
	txParams, err := h.buildSubAccountInitTx(&buildParams)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx params err: "+err.Error())
		return fmt.Errorf("buildSubAccountInitSubAccountTx err: %s", err.Error())
	}

	signList, _, err := h.buildTx(ctx, &paramBuildTx{
		txParams:   txParams,
		skipGroups: []int{},
		chainType:  req.chainType,
		address:    req.address,
		action:     common.DasActionEnableSubAccount,
		account:    req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp.Action = common.DasActionEnableSubAccount
	resp.SignKey = signList.SignKey
	resp.List = signList.List
	resp.SignList = signList.List[0].SignList
	apiResp.ApiRespOK(resp)
	return nil
}
