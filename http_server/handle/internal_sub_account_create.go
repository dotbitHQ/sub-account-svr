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
	"strings"
)

type RespInternalSubAccountCreate struct {
	TaskIds []string `json:"task_ids"`
}

func (h *HttpHandle) InternalSubAccountCreate(ctx *gin.Context) {
	var (
		funcName = "InternalSubAccountCreate"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountCreate
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

	if err = h.doInternalSubAccountCreate(&req, &apiResp); err != nil {
		log.Error("doInternalSubAccountCreate err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doInternalSubAccountCreate(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	var resp RespInternalSubAccountCreate
	resp.TaskIds = make([]string, 0)

	// check params
	if err := h.doSubAccountCheckParams(req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountCheckParams err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check account
	acc, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionCreateSubAccount)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check list
	isOk, respCheck, err := h.doSubAccountCheckList(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckList err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if !isOk {
		log.Error("doSubAccountCheckList:", toolib.JsonString(respCheck))
		apiResp.ApiRespErr(api_code.ApiCodeCreateListCheckFail, "create list check failed")
		return nil
	}

	// create
	if acc.ManagerChainType != config.Cfg.Server.ManagerChainType || !strings.EqualFold(acc.Manager, config.Cfg.Server.ManagerAddress) {
		apiResp.ApiRespErr(api_code.ApiCodeNotHaveManagementPermission, "not have management permission")
		return nil
	}
	// do distribution
	parentAccountId := acc.AccountId
	if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}
	taskList, taskMap, err := getTaskAndTaskMap(h.DasCore, req, parentAccountId, tables.TaskTypeDelegate)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("getTaskAndTaskMap err: %s", err.Error())
	}
	//
	for i, v := range taskList {
		records, _ := taskMap[v.TaskId]
		if err := h.DbDao.CreateTaskWithRecords(&taskList[i], records); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
			return fmt.Errorf("CreateTaskWithRecords err: %s", err.Error())
		}
		resp.TaskIds = append(resp.TaskIds, v.TaskId)
	}

	apiResp.ApiRespOK(resp)
	return nil
}
