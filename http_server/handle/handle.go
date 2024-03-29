package handle

import (
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/dao"
	"das_sub_account/lb"
	"das_sub_account/txtool"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/dascache"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/http_api/logger"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"net"
	"strings"
)

var (
	log = logger.NewLogger("http_handle", logger.LevelDebug)
)

type HttpHandle struct {
	Ctx           context.Context
	DasCore       *core.DasCore
	DasCache      *dascache.DasCache
	TxBuilderBase *txbuilder.DasTxBuilderBase
	DbDao         *dao.DbDao
	RC            *cache.RedisCache
	TxTool        *txtool.SubAccountTxTool
	LB            *lb.LoadBalancing
	SmtServerUrl  *string
	ServerScript  *types.Script
}

func GetClientIp(ctx *gin.Context) (string, string) {
	clientIP := fmt.Sprintf("%v", ctx.Request.Header.Get("X-Real-IP"))
	remoteAddrIP, _, _ := net.SplitHostPort(ctx.Request.RemoteAddr)
	return clientIP, remoteAddrIP
}

func (h *HttpHandle) checkSystemUpgrade(apiResp *api_code.ApiResp) error {
	if config.Cfg.Server.IsUpdate {
		apiResp.ApiRespErr(api_code.ApiCodeSystemUpgrade, api_code.TextSystemUpgrade)
		return fmt.Errorf("backend system upgrade")
	}
	ConfigCellDataBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsMain)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "ConfigCellDataBuilderByTypeArgs err")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
	}
	status, _ := ConfigCellDataBuilder.Status()
	if status != 1 {
		apiResp.ApiRespErr(api_code.ApiCodeSystemUpgrade, api_code.TextSystemUpgrade)
		return fmt.Errorf("contract system upgrade")
	}
	ok, err := h.DasCore.CheckContractStatusOK(common.DASContractNameSubAccountCellType)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "CheckContractStatusOK err")
		return fmt.Errorf("CheckContractStatusOK err: %s", err.Error())
	} else if !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSystemUpgrade, api_code.TextSystemUpgrade)
		return fmt.Errorf("contract system upgrade")
	}
	return nil
}

func doSendTransactionError(err error, apiResp *api_code.ApiResp) error {
	if strings.Contains(err.Error(), "PoolRejectedDuplicatedTransaction") ||
		strings.Contains(err.Error(), "Dead(OutPoint(") ||
		strings.Contains(err.Error(), "Unknown(OutPoint(") ||
		(strings.Contains(err.Error(), "getInputCell") && strings.Contains(err.Error(), "not live")) {

		apiResp.ApiRespErr(api_code.ApiCodeRejectedOutPoint, "SendTransaction err")
		return fmt.Errorf("SendTransaction err: %s", err.Error())
	}

	apiResp.ApiRespErr(api_code.ApiCodeError500, "send tx err")
	return fmt.Errorf("SendTransaction err: %s", err.Error())
}

func doApiError(err error, apiResp *api_code.ApiResp) {
	if strings.Contains(err.Error(), "PoolRejectedDuplicatedTransaction") ||
		strings.Contains(err.Error(), "Dead(OutPoint(") ||
		strings.Contains(err.Error(), "Unknown(OutPoint(") ||
		(strings.Contains(err.Error(), "getInputCell") && strings.Contains(err.Error(), "not live")) {

		apiResp.ApiRespErr(api_code.ApiCodeRejectedOutPoint, "send tx err")
	}
}

func doDasBalanceError(err error, apiResp *api_code.ApiResp) error {
	if err == core.ErrRejectedOutPoint {
		apiResp.ApiRespErr(api_code.ApiCodeRejectedOutPoint, "ErrRejectedOutPoint")
	} else if err == core.ErrNotEnoughChange {
		apiResp.ApiRespErr(api_code.ApiCodeNotEnoughChange, "ErrNotEnoughChange")
	} else if err == core.ErrInsufficientFunds {
		apiResp.ApiRespErr(api_code.ApiCodeInsufficientBalance, "ErrInsufficientFunds")
	} else {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "")
	}
	return err
}

type LBHttpHandle struct {
	Ctx context.Context
	RC  *cache.RedisCache
	LB  *lb.LoadBalancing
}
