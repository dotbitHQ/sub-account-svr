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
	"strings"
)

type ReqTransactionStatus struct {
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
	Action    common.DasAction `json:"action"`
	Account   string           `json:"account"`
}

type RespTransactionStatus struct {
	BlockNumber uint64   `json:"block_number"`
	Hash        string   `json:"hash"`
	Status      TxStatus `json:"status"`
}

type TxStatus int

const (
	TxStatusRejected  TxStatus = -1
	TxStatusPending   TxStatus = 0
	TxStatusCommitted TxStatus = 1
	TxStatusUnSend    TxStatus = 2
)

func (h *HttpHandle) TransactionStatus(ctx *gin.Context) {
	var (
		funcName = "TransactionStatus"
		clientIp = GetClientIp(ctx)
		req      ReqTransactionStatus
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

	if err = h.doTransactionStatus(&req, &apiResp); err != nil {
		log.Error("doTransactionStatus err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doTransactionStatus(req *ReqTransactionStatus, apiResp *api_code.ApiResp) error {
	var resp RespTransactionStatus

	// check params
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex

	// check account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil
	} else if (req.chainType != acc.OwnerChainType && !strings.EqualFold(req.address, acc.Owner)) ||
		(req.chainType != acc.ManagerChainType && !strings.EqualFold(req.address, acc.Manager)) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return nil
	}

	switch req.Action {
	case common.DasActionEnableSubAccount, common.DasActionCreateSubAccount,
		common.DasActionConfigSubAccountCustomScript:
		task, err := h.DbDao.GetTaskInfoByParentAccountIdWithAction(accountId, req.Action)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query task")
			return fmt.Errorf("GetTaskInfoByParentAccountIdWithAction: %s", err.Error())
		} else if task.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeTransactionNotExist, "not exist tx")
			return nil
		} else {
			switch task.TxStatus {
			case tables.TxStatusUnSend:
				resp.Status = TxStatusUnSend
			case tables.TxStatusPending:
				resp.Status = TxStatusPending
			default:
				apiResp.ApiRespErr(api_code.ApiCodeTransactionNotExist, "not exist tx")
				return nil
			}
		}
	case common.DasActionEditSubAccount:
		record, err := h.DbDao.GetLatestSmtRecordByAccountIdAction(accountId, req.Action)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query record")
			return fmt.Errorf("GetLatestSmtRecordByAccountIdAction: %s", err.Error())
		} else if record.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeTransactionNotExist, "not exist tx")
			return nil
		} else if record.TaskId == "" {
			resp.Status = TxStatusUnSend
		} else {
			task, err := h.DbDao.GetTaskByTaskId(record.TaskId)
			if err != nil {
				apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to task record")
				return fmt.Errorf("GetLatestSmtRecordByAccountIdAction: %s", err.Error())
			} else {
				switch task.TxStatus {
				case tables.TxStatusUnSend:
					resp.Status = TxStatusUnSend
				case tables.TxStatusPending:
					resp.Status = TxStatusPending
				default:
					apiResp.ApiRespErr(api_code.ApiCodeTransactionNotExist, "not exist tx")
					return nil
				}
			}
		}
	default:
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("not exist action[%s]", req.Action))
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
