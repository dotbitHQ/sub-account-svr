package handle

import (
	"das_sub_account/config"
	"das_sub_account/tables"
	"encoding/json"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
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
	MintForAccount string                  `json:"mint_for_account"`
	AccountCharStr []common.AccountCharSet `json:"account_char_str"`
	RegisterYears  uint64                  `json:"register_years"`
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
}

type RespSubAccountCreate struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountCreateNew(ctx *gin.Context) {
	var (
		funcName               = "SubAccountCreateNew"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountCreate
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

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
	acc, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionUpdateSubAccount)
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

	// check balance
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "get config cell err")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	newSubAccountPrice, _ := molecule.Bytes2GoU64(configCellBuilder.ConfigCellSubAccount.NewSubAccountPrice().RawData())
	totalCapacity := uint64(0)
	totalRegisterYears := uint64(0)
	for _, v := range req.SubAccountList {
		totalRegisterYears += v.RegisterYears
	}

	quoteCell, err := h.DasCore.GetQuoteCell()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "failed to get quote cell")
		return fmt.Errorf("GetQuoteCell err: %s", err.Error())
	}
	totalCapacity = config.PriceToCKB(newSubAccountPrice, quoteCell.Quote(), totalRegisterYears)
	//totalCapacity = totalRegisterYears * newSubAccountPrice

	_, _, err = h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          nil,
		LockScript:        balanceDasLock,
		CapacityNeed:      totalCapacity,
		CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		return doDasBalanceError(err, apiResp)
	}

	// get mint sign info
	minSignInfo, listSmtRecord, err := h.doMinSignInfo(parentAccountId, acc, req, apiResp)
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
		MinSignInfo:     minSignInfo,
		ListSmtRecord:   listSmtRecord,
	}
	signData := dataCache.GetCreateSignData(acc.ManagerAlgorithmId, apiResp)
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
	resp.SubAction = common.SubActionCreate
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

func (h *HttpHandle) doMinSignInfo(parentAccountId string, acc *tables.TableAccountInfo, req *ReqSubAccountCreate, apiResp *api_code.ApiResp) (*tables.TableMintSignInfo, []tables.TableSmtRecordInfo, error) {
	expiredAt := uint64(time.Now().Add(time.Hour * 24 * 7).Unix())
	if expiredAt > acc.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return nil, nil, fmt.Errorf("account expiring soon")
	}

	var listSmtRecord []tables.TableSmtRecordInfo
	var listKeyValue []tables.MintSignInfoKeyValue

	tree := smt.NewSmtSrv(*h.SmtServerUrl, "")
	var smtKv []smt.SmtKv
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
			MintType:        tables.MintTypeManual,
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
		smtKv = append(smtKv, smt.SmtKv{
			smtKey,
			smtValue,
		})

		listKeyValue = append(listKeyValue, tables.MintSignInfoKeyValue{
			Key:   subAccountId,
			Value: common.Bytes2Hex(registerArgs),
		})
	}
	opt := smt.SmtOpt{GetProof: false, GetRoot: true}
	r, err := tree.UpdateSmt(smtKv, opt)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt update err")
		return nil, nil, fmt.Errorf("tree.Update err: %s", err.Error())
	}
	root := r.Root
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt root err")
		return nil, nil, fmt.Errorf("tree.Root err: %s", err.Error())
	}
	keyValueStr, _ := json.Marshal(&listKeyValue)
	minSignInfo := tables.TableMintSignInfo{
		SmtRoot:   common.Bytes2Hex(root),
		ExpiredAt: expiredAt,
		Timestamp: uint64(time.Now().UnixNano() / 1e6),
		KeyValue:  string(keyValueStr),
		ChainType: acc.ManagerChainType,
		Address:   acc.Manager,
		SubAction: common.SubActionCreate,
	}
	minSignInfo.InitMintSignId(parentAccountId)
	for i, _ := range listSmtRecord {
		listSmtRecord[i].MintSignId = minSignInfo.MintSignId
	}
	return &minSignInfo, listSmtRecord, nil
}
