package handle

import (
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
)

type ReqSmtCheck struct {
	ParentAccountId string `json:"parent_account_id"`
	Limit           int    `json:"limit" yaml:"limit"`
}

type RespSmtCheck struct {
	List []SmtCheckData `json:"list"`
}

type SmtCheckData struct {
	AccountId  string `json:"account_id"`
	ChainValue string `json:"chain_value"`
	SmtValue   string `json:"smt_value"`
	Diff       bool   `json:"diff"`
}

func (h *HttpHandle) SmtCheck(ctx *gin.Context) {
	var (
		funcName = "SmtCheck"
		clientIp = GetClientIp(ctx)
		req      ReqSmtCheck
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

	if err = h.doSmtCheck(&req, &apiResp); err != nil {
		log.Error("doSmtCheck err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSmtCheck(req *ReqSmtCheck, apiResp *api_code.ApiResp) error {
	var resp RespSmtCheck
	resp.List = make([]SmtCheckData, 0)

	// task
	if req.Limit <= 0 {
		req.Limit = 1
	}

	taskList, err := h.DbDao.GetLatestTaskByParentAccountId(req.ParentAccountId, req.Limit)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetLatestTaskByParentAccountId err: %s", err.Error())
	}

	var taskIds []string
	for _, v := range taskList {
		taskIds = append(taskIds, v.TaskId)
	}

	// records
	records, err := h.DbDao.GetSmtRecordListByTaskIds(taskIds, tables.RecordTypeChain)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSmtRecordListByTaskIds err: %s", err.Error())
	}

	var subAccountIds []string
	for _, v := range records {
		subAccountIds = append(subAccountIds, v.AccountId)
	}

	// smt
	smtList, err := h.DbDao.GetSmtInfoBySubAccountIds(subAccountIds)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSmtInfoBySubAccountIds err: %s", err.Error())
	}

	// chain
	var subAccountBuilderMap = make(map[string]*witness.SubAccountBuilder)
	for _, v := range taskList {
		outpoint := common.String2OutPointStruct(v.Outpoint)
		res, err := h.DasCore.Client().GetTransaction(h.Ctx, outpoint.TxHash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("GetTransaction err: %s", err.Error())
		}
		builderMap, err := witness.SubAccountBuilderMapFromTx(res.Transaction)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("SubAccountBuilderMapFromTx err: %s", err.Error())
		}
		for k, _ := range builderMap {
			if _, ok := subAccountBuilderMap[k]; !ok {
				subAccountBuilderMap[k] = builderMap[k]
			}
		}
	}

	// check
	for _, v := range smtList {
		item, _ := subAccountBuilderMap[v.AccountId]
		chainValue := item.CurrentSubAccount.ToH256()
		diff := common.Bytes2Hex(chainValue) != v.LeafDataHash
		resp.List = append(resp.List, SmtCheckData{
			AccountId:  v.AccountId,
			ChainValue: common.Bytes2Hex(chainValue),
			SmtValue:   v.LeafDataHash,
			Diff:       diff,
		})
	}

	apiResp.ApiRespOK(resp)
	return nil
}
