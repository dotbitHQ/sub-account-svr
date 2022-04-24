package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqTaskInfo struct {
	TaskId string `json:"task_id"`
	Hash   string `json:"hash"`
}

type RespTaskInfo struct {
	Status TaskStatus `json:"status"`
}

type TaskStatus int

const (
	TaskStatusPending TaskStatus = 0
	TaskStatusOk      TaskStatus = 1
	TaskStatusFail    TaskStatus = 2
)

func (h *HttpHandle) TaskInfo(ctx *gin.Context) {
	var (
		funcName = "TaskInfo"
		clientIp = GetClientIp(ctx)
		req      ReqTaskInfo
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

	if err = h.doTaskInfo(&req, &apiResp); err != nil {
		log.Error("doTaskInfo err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doTaskInfo(req *ReqTaskInfo, apiResp *api_code.ApiResp) error {
	var resp RespTaskInfo

	var task tables.TableTaskInfo
	var err error
	if req.TaskId != "" {
		task, err = h.DbDao.GetTaskByTaskId(req.TaskId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "search task err")
			return fmt.Errorf("GetTaskByTaskId err: %s", err.Error())
		}
	} else {
		task, err = h.DbDao.GetTaskByHash(req.Hash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "search task err")
			return fmt.Errorf("GetTaskByHash err: %s", err.Error())
		}
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
	}

	switch task.SmtStatus {
	case tables.SmtStatusRollbackComplete:
		resp.Status = TaskStatusFail
	}

	apiResp.ApiRespOK(resp)
	return nil
}
