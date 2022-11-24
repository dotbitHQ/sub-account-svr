package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

func (h *HttpHandle) SubAccountCreateNew(ctx *gin.Context) {
	var (
		funcName = "SubAccountCreateNew"
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

	if err = h.doSubAccountCreateNew(&req, &apiResp); err != nil {
		log.Error("doSubAccountCreateNew err:", err.Error(), funcName, clientIp)
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountCreateNew(req *ReqSubAccountCreate, apiResp *api_code.ApiResp) error {
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

	// das lock
	var balanceDasLock, balanceDasType *types.Script
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
	log.Info("doSubAccountCreateNew:", balanceDasLock, balanceDasType)

	// do distribution
	parentAccountId := acc.AccountId
	if _, ok := config.Cfg.SuspendMap[parentAccountId]; ok {
		apiResp.ApiRespErr(api_code.ApiCodeSuspendOperation, "suspend operation")
		return nil
	}

	// todo check balance

	// get mint sign info
	minSignInfo, listSmtRecord, err := h.doMinSignInfo(parentAccountId, acc.ExpiredAt, req, apiResp)
	if err != nil {
		return fmt.Errorf("doMinSignInfo err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	log.Info("doSubAccountCreateNew:", parentAccountId, minSignInfo.ExpiredAt, len(listSmtRecord))

	// sign info
	dataCache := UpdateSubAccountCache{
		ParentAccountId: acc.AccountId,
		Account:         req.Account,
		ChainType:       req.chainType,
		Address:         req.address,
		SubAction:       common.SubActionCreate,
		OldSignMsg:      "",
		MinSignInfo:     *minSignInfo,
		ListSmtRecord:   listSmtRecord,
	}
	signData := dataCache.GetCreateSignData(acc, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	dataCache.OldSignMsg = signData.SignMsg // for check after user sign

	// cache
	signKey := dataCache.CacheKey()
	cacheStr := toolib.JsonString(&dataCache)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	// resp
	resp.Action = common.DasActionUpdateSubAccount
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		//SignKey: "",
		SignList: []txbuilder.SignData{
			signData,
		},
	})

	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doMinSignInfo(parentAccountId string, accExpiredAt uint64, req *ReqSubAccountCreate, apiResp *api_code.ApiResp) (*tables.TableMintSignInfo, []tables.TableSmtRecordInfo, error) {
	expiredAt := uint64(time.Now().Add(time.Hour * 24 * 7).Unix())
	if expiredAt > accExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "expires soon")
		return nil, nil, fmt.Errorf("expires soon")
	}

	var listSmtRecord []tables.TableSmtRecordInfo
	tree := smt.NewSparseMerkleTree(nil)
	for _, v := range req.SubAccountList {
		subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
		ownerHex := core.DasAddressHex{
			DasAlgorithmId: v.chainType.ToDasAlgorithmId(true),
			AddressHex:     v.address,
			IsMulti:        false,
			ChainType:      v.chainType,
		}
		registerArgs, err := h.DasCore.Daf().HexToArgs(ownerHex, ownerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "HexToArgs err")
			return nil, nil, fmt.Errorf("HexToArgs err: %s", err.Error())
		}
		var content []byte
		content, err = json.Marshal(v.AccountCharStr)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "AccountCharStr err")
			return nil, nil, fmt.Errorf("json Marshal err: %s", err.Error())
		}
		tmp := tables.TableSmtRecordInfo{
			SvrName:         config.Cfg.Slb.SvrName,
			AccountId:       subAccountId,
			Nonce:           0,
			RecordType:      tables.RecordTypeDefault,
			TaskId:          "",
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			Content:         string(content),
			RegisterYears:   v.RegisterYears,
			RegisterArgs:    common.Bytes2Hex(registerArgs),
			EditKey:         "",
			Signature:       "",
			EditArgs:        "",
			RenewYears:      0,
			EditRecords:     "",
			Timestamp:       time.Now().UnixNano() / 1e6,
			SubAction:       common.SubActionCreate,
			MintSignId:      "",
		}
		listSmtRecord = append(listSmtRecord, tmp)

		smtKey := smt.AccountIdToSmtH256(subAccountId)
		smtValue, err := blake2b.Blake256(registerArgs)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt value err")
			return nil, nil, fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
		}
		if err = tree.Update(smtKey, smtValue); err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt update err")
			return nil, nil, fmt.Errorf("tree.Update err: %s", err.Error())
		}
	}
	root, err := tree.Root()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt root err")
		return nil, nil, fmt.Errorf("tree.Root err: %s", err.Error())
	}
	minSignInfo := tables.TableMintSignInfo{
		SmtRoot:    common.Bytes2Hex(root),
		ExpiredAt:  expiredAt,
		MintSignId: "",
		Signature:  "",
		Timestamp:  uint64(time.Now().UnixNano() / 1e6),
	}
	minSignInfo.InitMintSignId(parentAccountId)
	for i, _ := range listSmtRecord {
		listSmtRecord[i].MintSignId = minSignInfo.MintSignId
	}
	return &minSignInfo, listSmtRecord, nil
}
