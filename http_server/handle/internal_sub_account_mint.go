package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/gin-gonic/gin"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type RespInternalSubAccountMint struct {
}

func (h *HttpHandle) InternalSubAccountMint(ctx *gin.Context) {
	var (
		funcName = "InternalSubAccountMint"
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

	if err = h.doInternalSubAccountMint(&req, &apiResp); err != nil {
		log.Error("doInternalSubAccountMint err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doInternalSubAccountMint(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	var resp RespInternalSubAccountMint

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

	recordList, err := getRecordList(h.DasCore.Daf(), req, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("getRecordList err: %s", err.Error())
	}
	if err := h.DbDao.CreateSmtRecordList(recordList); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, err.Error())
		return fmt.Errorf("CreateSmtRecordList err: %s", err.Error())
	}

	apiResp.ApiRespOK(resp)
	return nil
}

func getRecordList(daf *core.DasAddressFormat, req *ReqSubAccountCreate, parentAccountId string) ([]tables.TableSmtRecordInfo, error) {
	var recordList []tables.TableSmtRecordInfo

	for _, v := range req.SubAccountList {
		subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: v.chainType.ToDasAlgorithmId(true),
			AddressHex:     v.address,
			IsMulti:        false,
			ChainType:      v.chainType,
		}
		registerArgs, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			return nil, fmt.Errorf("HexToArgs err: %s", err.Error())
		}

		tmpRecord := tables.TableSmtRecordInfo{
			Id:              0,
			AccountId:       subAccountId,
			Nonce:           0,
			RecordType:      tables.RecordTypeDefault,
			TaskId:          "",
			Action:          common.DasActionCreateSubAccount,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			RegisterYears:   v.RegisterYears,
			RegisterArgs:    common.Bytes2Hex(registerArgs),
			EditKey:         "",
			Signature:       "",
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       time.Now().UnixNano() / 1e6,
		}
		recordList = append(recordList, tmpRecord)
	}
	return recordList, nil
}
