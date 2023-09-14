package handle

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqSubAccountMintStatus struct {
	SubAccount string `json:"sub_account"`
}

type RespSubAccountMintStatus struct {
	Status TaskStatus `json:"status"`
}
type TaskStatus int

const (
	TaskStatusPending TaskStatus = 0
	TaskStatusOk      TaskStatus = 1
	TaskStatusFail    TaskStatus = 2
)

func (h *HttpHandle) SubAccountMintStatus(ctx *gin.Context) {
	var (
		funcName               = "SubAccountMintStatus"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountMintStatus
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

	if err = h.doSubAccountMintStatus(&req, &apiResp); err != nil {
		log.Error("doSubAccountMintStatus err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountMintStatus(req *ReqSubAccountMintStatus, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountMintStatus

	if req.SubAccount == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "sub account is nil")
		return nil
	}
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.SubAccount))

	record, err := h.DbDao.GetLatestMintRecord(accountId, common.DasActionUpdateSubAccount, common.SubActionCreate)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "get mint record err")
		return fmt.Errorf("GetLatestMintRecord err: %s", err.Error())
	}
	if record.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeTaskNotExist, "task not exist")
		return nil
	}
	if record.TaskId == "" {
		resp.Status = TaskStatusPending
		apiResp.ApiRespOK(resp)
		return nil
	}

	task, err := h.DbDao.GetTaskByTaskId(record.TaskId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "search task err")
		return fmt.Errorf("GetTaskByTaskId err: %s", err.Error())
	}

	if task.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeTaskNotExist, "task not exist")
		return nil
	}

	resp.Status = TaskStatusPending
	switch task.TxStatus {
	case tables.TxStatusCommitted:
		resp.Status = TaskStatusOk
	case tables.TxStatusRejected:
		resp.Status = TaskStatusFail
		if task.TaskType == tables.TaskTypeDelegate && task.Retry < config.Cfg.Das.MaxRetry {
			resp.Status = TaskStatusPending
		}
	}

	switch task.SmtStatus {
	case tables.SmtStatusRollbackComplete:
		resp.Status = TaskStatusFail
	}

	apiResp.ApiRespOK(resp)
	return nil
}
