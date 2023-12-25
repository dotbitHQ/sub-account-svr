package handle

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

type ReqTransactionStatus struct {
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
	Action    common.DasAction `json:"action"`
	SubAction common.SubAction `json:"sub_action"`
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
		funcName               = "TransactionStatus"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqTransactionStatus
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doTransactionStatus(&req, &apiResp); err != nil {
		log.Error("doTransactionStatus err:", err.Error(), funcName, clientIp, ctx)
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
	}
	if (req.chainType != acc.OwnerChainType && !strings.EqualFold(req.address, acc.Owner)) &&
		(req.chainType != acc.ManagerChainType && !strings.EqualFold(req.address, acc.Manager)) &&
		(req.Action != common.DasActionFulfillApproval && req.Action != common.DasActionUpdateSubAccount ||
			req.Action == common.DasActionUpdateSubAccount && req.SubAction != common.SubActionFullfillApproval) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return nil
	}

	switch req.Action {
	case common.DasActionEnableSubAccount, common.DasActionConfigSubAccountCustomScript,
		common.DasActionCollectSubAccountProfit, common.DasActionConfigSubAccount:
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
	case common.DasActionUpdateSubAccount:
		var record tables.TableSmtRecordInfo
		switch req.SubAction {
		case common.SubActionCreate:
			record, err = h.DbDao.GetLatestSmtRecordByParentAccountIdAction(accountId, req.Action, req.SubAction)
		case common.SubActionEdit:
			record, err = h.DbDao.GetLatestSmtRecordByAccountIdAction(accountId, req.Action, req.SubAction)
		case common.SubActionRenew:
			record, err = h.DbDao.GetLatestSmtRecordByParentAccountIdAction(accountId, req.Action, req.SubAction)
		case common.SubActionCreateApproval, common.SubActionDelayApproval, common.SubActionRevokeApproval, common.SubActionFullfillApproval:
			record, err = h.DbDao.GetLatestSmtRecordByAccountIdAction(accountId, req.Action, req.SubAction)
		default:
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("not exist sub-action[%s]", req.SubAction))
			return nil
		}
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
	case common.DasActionCreateApproval, common.DasActionDelayApproval,
		common.DasActionRevokeApproval, common.DasActionFulfillApproval:
		actionList := []common.DasAction{req.Action}
		tx, err := h.DbDao.GetPendingStatus(req.chainType, req.address, actionList)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "search tx status err")
			return fmt.Errorf("GetTransactionStatus err: %s", err.Error())
		}
		if tx.Id == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeTransactionNotExist, "not exits tx")
			return nil
		}
		resp.BlockNumber = tx.BlockNumber
		resp.Hash, _ = common.String2OutPoint(tx.Outpoint)
		resp.Status = TxStatus(tx.Status)
	default:
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("not exist action[%s]", req.Action))
		return nil
	}

	apiResp.ApiRespOK(resp)
	return nil
}
