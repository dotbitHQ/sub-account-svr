package handle

import (
	"bytes"
	"context"
	"das_sub_account/cache"
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"das_sub_account/txtool"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqSubAccountCreate struct {
	core.ChainTypeAddress
	chainType      common.ChainType
	address        string
	Account        string             `json:"account"`
	SubAccountList []CreateSubAccount `json:"sub_account_list"`
}

type CreateSubAccount struct {
	Account        string                  `json:"account"`
	AccountCharStr []common.AccountCharSet `json:"account_char_str"`
	RegisterYears  uint64                  `json:"register_years"`
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
}

type RespSubAccountCreate struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountCreate(ctx *gin.Context) {
	var (
		funcName = "SubAccountCreate"
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

	if err = h.doSubAccountCreate(&req, &apiResp); err != nil {
		log.Error("doSubAccountCreate err:", err.Error(), funcName, clientIp)
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountCreate(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountCreate

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

	// check custom-script
	subAccountLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("GetSubAccountCell err: %s", err.Error())
	}
	subDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	if subDetail.HasCustomScriptArgs() {
		apiResp.ApiRespErr(api_code.ApiCodeCustomScriptSet, "custom-script set")
		return nil
	}
	//if err := h.doSubAccountCheckCustomScript(acc.AccountId, req, apiResp); err != nil {
	//	return fmt.Errorf("doSubAccountCheckCustomScript err: %s", err.Error())
	//} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
	//	return nil
	//}

	// das lock
	var balanceDasLock, balanceDasType *types.Script
	//if acc.OwnerChainType == req.chainType && strings.EqualFold(acc.Owner, req.address) {
	//	balanceDasLock, balanceDasType, err = h.DasCore.FormatAddressToDasLockScript(acc.OwnerChainType, acc.Owner, true)
	//	if err != nil {
	//		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
	//		return fmt.Errorf("FormatAddressToDasLockScript err: %s", err.Error())
	//	}
	//} else
	if acc.ManagerChainType == req.chainType && strings.EqualFold(acc.Manager, req.address) {
		balanceDasLock, balanceDasType, err = h.DasCore.Daf().HexToScript(core.DasAddressHex{
			DasAlgorithmId: acc.ManagerChainType.ToDasAlgorithmId(true),
			AddressHex:     acc.Manager,
			IsMulti:        false,
			ChainType:      acc.ManagerChainType,
		})
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("FormatAddressToDasLockScript err: %s", err.Error())
		}
	} else {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return nil
	}

	// create
	// do distribution
	parentAccountId := acc.AccountId
	if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}
	taskList, taskMap, err := getTaskAndTaskMap(h.DasCore.Daf(), req, parentAccountId, tables.TaskTypeNormal)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("getTaskAndTaskMap err: %s", err.Error())
	}
	// do check
	resCheck, err := h.TxTool.DoCheckBeforeBuildTx(parentAccountId)
	if err != nil {
		if resCheck != nil && resCheck.Continue {
			apiResp.ApiRespErr(api_code.ApiCodeTaskInProgress, "task in progress")
			return nil
		}
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("DoCheckBeforeBuildTx err: %s", err.Error())
	}

	// get smt tree
	mongoStore := smt.NewMongoStore(h.Ctx, h.Mongo, config.Cfg.DB.Mongo.SmtDatabase, parentAccountId)
	tree := smt.NewSparseMerkleTree(mongoStore)

	// check root
	currentRoot, _ := tree.Root()
	subDataDetail := witness.ConvertSubAccountCellOutputData(resCheck.SubAccountLiveCell.OutputData)
	log.Warn("Compare root:", parentAccountId, common.Bytes2Hex(currentRoot), common.Bytes2Hex(subDataDetail.SmtRoot))
	if bytes.Compare(currentRoot, subDataDetail.SmtRoot) != 0 {
		apiResp.ApiRespErr(api_code.ApiCodeSmtDiff, "smt root diff")
		return nil
	}

	// lock smt and unlock
	if err := h.RC.LockWithRedis(parentAccountId); err != nil {
		if err == cache.ErrDistributedLockPreemption {
			apiResp.ApiRespErr(api_code.ApiCodeDistributedLockPreemption, err.Error())
		} else {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		}
		return fmt.Errorf("LockWithRedis err: %s", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := h.RC.UnLockWithRedis(parentAccountId); err != nil {
			fmt.Println("UnLockWithRedis:", err.Error())
		}
		cancel()
	}()
	h.RC.DoLockExpire(ctx, parentAccountId)

	// build tx
	res, err := h.TxTool.BuildTxs(&txtool.ParamBuildTxs{
		TaskList:             taskList,
		TaskMap:              taskMap,
		Account:              acc,
		SubAccountLiveCell:   resCheck.SubAccountLiveCell,
		Tree:                 tree,
		BaseInfo:             resCheck.BaseInfo,
		BalanceDasLock:       balanceDasLock,
		BalanceDasType:       balanceDasType,
		SubAccountBuilderMap: nil,
		SubAccountValueMap:   nil,
		SubAccountIds:        nil,
	})
	if err != nil {
		return doBuildTxs(err, apiResp)
	}

	// sign info
	signInfoCacheList := SignInfoCacheList{
		Action:        common.DasActionCreateSubAccount,
		Account:       req.Account,
		TaskIdList:    nil,
		BuilderTxList: nil,
	}

	for i, _ := range taskList {
		var skipGroups []int
		skipGroups = []int{1} // skip sub-account-cell
		if res.IsCustomScript {
			skipGroups = []int{0}
		}
		log.Info("skipGroups:", res.DasTxBuilderList[i].ServerSignGroup)
		signList, err := res.DasTxBuilderList[i].GenerateDigestListFromTx(skipGroups)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return fmt.Errorf("GenerateDigestListFromTx err: %s", err.Error())
		}
		signInfoCacheList.TaskIdList = append(signInfoCacheList.TaskIdList, taskList[i].TaskId)
		signInfoCacheList.BuilderTxList = append(signInfoCacheList.BuilderTxList, res.DasTxBuilderList[i].DasTxBuilderTransaction)
		signInfo := SignInfo{
			//SignKey:  "",
			SignList: signList,
		}
		resp.List = append(resp.List, signInfo)
	}

	// do cache
	resp.SignKey = signInfoCacheList.SignKey()
	cacheStr := toolib.JsonString(&signInfoCacheList)
	if err = h.RC.SetSignTxCache(resp.SignKey, cacheStr); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	log.Info("doSubAccountCreate:", toolib.JsonString(resp))
	resp.Action = common.DasActionCreateSubAccount
	apiResp.ApiRespOK(resp)
	return nil
}

func getTaskAndTaskMap(daf *core.DasAddressFormat, req *ReqSubAccountCreate, parentAccountId string, taskType tables.TaskType) ([]tables.TableTaskInfo, map[string][]tables.TableSmtRecordInfo, error) {
	var taskList []tables.TableTaskInfo
	var taskMap = make(map[string][]tables.TableSmtRecordInfo)
	taskId, count := "", 0
	for _, v := range req.SubAccountList {
		if count == config.Cfg.Das.MaxCreateCount {
			count = 0
		}
		if count == 0 {
			tmpTask := tables.TableTaskInfo{
				Id:              0,
				TaskId:          "",
				TaskType:        taskType,
				ParentAccountId: parentAccountId,
				Action:          common.DasActionCreateSubAccount,
				RefOutpoint:     "",
				BlockNumber:     0,
				Outpoint:        "",
				Timestamp:       time.Now().UnixNano() / 1e6,
				SmtStatus:       tables.SmtStatusNeedToWrite,
				TxStatus:        tables.TxStatusUnSend,
			}
			tmpTask.InitTaskId()
			taskId = tmpTask.TaskId
			taskList = append(taskList, tmpTask)
		}
		count++
		subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))

		ownerHex := core.DasAddressHex{
			DasAlgorithmId: v.chainType.ToDasAlgorithmId(true),
			AddressHex:     v.address,
			IsMulti:        false,
			ChainType:      v.chainType,
		}
		registerArgs, err := daf.HexToArgs(ownerHex, ownerHex)
		if err != nil {
			return nil, nil, fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		var content []byte
		if len(v.AccountCharStr) > 0 {
			content, err = json.Marshal(v.AccountCharStr)
			if err != nil {
				return nil, nil, fmt.Errorf("json Marshal err: %s", err.Error())
			}
		}
		tmpRecord := tables.TableSmtRecordInfo{
			Id:              0,
			AccountId:       subAccountId,
			Nonce:           0,
			RecordType:      tables.RecordTypeDefault,
			TaskId:          taskId,
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
			Content:         string(content),
		}
		taskMap[taskId] = append(taskMap[taskId], tmpRecord)
	}
	return taskList, taskMap, nil
}
