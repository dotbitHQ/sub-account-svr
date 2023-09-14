package handle

import (
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"time"
)

type ReqRecycleAccount struct {
	SubAccountIds []string `json:"sub_account_ids"`
}

type RespRecycleAccount struct {
}

func (h *HttpHandle) RecycleAccount(ctx *gin.Context) {
	var (
		funcName               = "RecycleAccount"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqRecycleAccount
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, toolib.JsonString(req), ctx)

	//time.Sleep(time.Minute * 3)
	if err = h.doRecycleAccount(&req, &apiResp); err != nil {
		log.Error("doRecycleAccount err:", err.Error(), funcName, clientIp, ctx)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doRecycleAccount(req *ReqRecycleAccount, apiResp *api_code.ApiResp) error {
	var resp RespRecycleAccount

	accConfigCell, err := h.DasCore.ConfigCellDataBuilderByTypeArgs(common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get config cell")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgs err: %s", err.Error())
	}
	expirationGracePeriod, err := accConfigCell.ExpirationGracePeriod()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get ExpirationGracePeriod")
		return fmt.Errorf("ExpirationGracePeriod err: %s", err.Error())
	}
	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "Failed to get TimeCell")
		return fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	timestamp := timeCell.Timestamp()
	log.Info("recycleSubAccount:", timestamp, expirationGracePeriod, timestamp-int64(expirationGracePeriod))
	timestamp = timestamp - int64(expirationGracePeriod)

	if len(req.SubAccountIds) == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid: SubAccountIds is nil")
		return nil
	}

	list, err := h.DbDao.GetAccountListByAccountIds(req.SubAccountIds)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get accounts")
		return fmt.Errorf("GetAccountListByAccountIds err: %s", err.Error())
	}

	var smtRecordList []tables.TableSmtRecordInfo
	for _, v := range list {
		if v.ParentAccountId == "" {
			continue
		}
		if v.ExpiredAt >= uint64(timestamp) {
			continue
		}
		smtRecord, err := h.DbDao.GetRecycleSmtRecord(v.AccountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to get smt record")
			return fmt.Errorf("GetRecycleSmtRecord err: %s", err.Error())
		} else if smtRecord.Id != 0 {
			continue
		}
		tmpSmtRecord := tables.TableSmtRecordInfo{
			SvrName:         "",
			AccountId:       v.AccountId,
			Nonce:           v.Nonce + 1,
			RecordType:      tables.RecordTypeDefault,
			MintType:        tables.MintTypeDefault,
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: v.ParentAccountId,
			Account:         v.Account,
			Timestamp:       time.Now().UnixMilli(),
			SubAction:       common.SubActionRecycle,
		}
		smtRecordList = append(smtRecordList, tmpSmtRecord)
	}
	log.Info("smtRecordList:", len(smtRecordList))
	if len(smtRecordList) > 0 {
		if err := h.DbDao.CreateRecycleSmtRecordList(smtRecordList); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to create smt record")
			return fmt.Errorf("CreateRecycleSmtRecordList err: %s", err.Error())
		}
	}

	apiResp.ApiRespOK(resp)
	return nil
}
