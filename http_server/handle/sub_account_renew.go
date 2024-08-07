package handle

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/tables"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/crypto/blake2b"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqSubAccountRenew struct {
	core.ChainTypeAddress
	chainType      common.ChainType
	address        string
	Account        string            `json:"account" binging:"required"`
	SubAccountList []RenewSubAccount `json:"sub_account_list" binding:"min=1"`
}

type RenewSubAccount struct {
	Account    string `json:"account"`
	RenewYears uint64 `json:"renew_years"`
}

type RespSubAccountRenew struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountRenew(ctx *gin.Context) {
	var (
		funcName               = "SubAccountRenew"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountRenew
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doSubAccountRenew(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doSubAccountRenew err:", err.Error(), funcName, clientIp, ctx.Request.Context())
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountRenew(ctx context.Context, req *ReqSubAccountRenew, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountRenew
	req.Account = strings.ToLower(req.Account)

	// check params
	if err := h.doSubAccountRenewCheckParams(req, apiResp); err != nil {
		return fmt.Errorf("doSubAccountCheckParams err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check account
	acc, err := h.doSubAccountRenewCheckAccount(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}

	// check list
	isOk, respCheck, err := h.doSubAccountRenewCheckList(req, apiResp)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckList err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	} else if !isOk {
		log.Error(ctx, "doSubAccountRenewCheckList:", toolib.JsonString(respCheck))
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

	if acc.ManagerChainType != req.chainType || !strings.EqualFold(acc.Manager, req.address) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return nil
	}
	parentAccountId := acc.AccountId

	// check manager dp cell have enough amount to pay
	configCellBuilder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "get config cell err")
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}
	renewSubAccountPrice, err := configCellBuilder.RenewSubAccountPrice()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "get config cell err")
		return fmt.Errorf("RenewSubAccountPrice err: %s", err.Error())
	}
	totalRenewYears := uint64(0)
	for _, v := range req.SubAccountList {
		totalRenewYears += v.RenewYears
	}
	totalPrice := renewSubAccountPrice * totalRenewYears

	lock, _, err := req.FormatChainTypeAddressToScript(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "FormatChainTypeAddressToScript err")
		return err
	}
	if _, _, _, err := h.DasCore.GetDpCells(&core.ParamGetDpCells{
		DasCache:    h.DasCache,
		LockScript:  lock,
		AmountNeed:  totalPrice,
		SearchOrder: indexer.SearchOrderAsc,
	}); err != nil {
		return doDasBalanceError(err, apiResp)
	}

	addressHex := &core.DasAddressHex{}
	addressHex.DasAlgorithmId = acc.ManagerChainType.ToDasAlgorithmId(true)
	addressHex.AddressHex = acc.Manager
	addressHex.ChainType = acc.ManagerChainType

	// get renew sign info
	listSmtRecord, renewSignInfo, err := h.doRenewSignInfo(common.ParamManager, *addressHex, acc, req, apiResp)
	if err != nil {
		return fmt.Errorf("doMinSignInfo err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	log.Info(ctx, "doRenewSignInfo:", parentAccountId, renewSignInfo, len(listSmtRecord))

	// sign info
	dataCache := UpdateSubAccountCache{
		ParentAccountId: parentAccountId,
		Account:         req.Account,
		ChainType:       addressHex.ChainType,
		AlgId:           addressHex.DasAlgorithmId,
		Address:         addressHex.AddressHex,
		SubAction:       common.SubActionRenew,
		MinSignInfo:     renewSignInfo,
		ListSmtRecord:   listSmtRecord,
	}
	signData := dataCache.GetCreateSignData(addressHex.DasAlgorithmId, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	dataCache.OldSignMsg = signData.SignMsg

	// cache
	signKey := dataCache.CacheKey()
	cacheStr := toolib.JsonString(&dataCache)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	// resp
	resp.Action = common.DasActionUpdateSubAccount
	resp.SubAction = common.SubActionRenew
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		SignList: []txbuilder.SignData{
			signData,
		},
	})
	resp.SignList = []txbuilder.SignData{
		signData,
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doSubAccountRenewCheckParams(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) error {
	if len(req.SubAccountList) > config.Cfg.Das.MaxCreateCount {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, fmt.Sprintf("more than max renew num %d", config.Cfg.Das.MaxCreateCount))
		return nil
	}
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex
	return nil
}

func (h *HttpHandle) doSubAccountRenewCheckAccount(req *ReqSubAccountRenew, apiResp *api_code.ApiResp) (*tables.TableAccountInfo, error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
		return nil, fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
	}
	if acc.Id == 0 {
		err = errors.New("account not exist")
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, err.Error())
		return nil, err
	}
	if acc.EnableSubAccount != tables.AccountEnableStatusOn {
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "sub account uninitialized")
		return nil, nil
	}
	if acc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account expired")
		return nil, nil
	}
	return &acc, nil
}

func (h *HttpHandle) doRenewSignInfo(signRole string, addressHex core.DasAddressHex, acc *tables.TableAccountInfo, req *ReqSubAccountRenew, apiResp *api_code.ApiResp) ([]tables.TableSmtRecordInfo, *tables.TableMintSignInfo, error) {
	nowTime := time.Now()

	expiredAt := uint64(nowTime.Add(time.Hour * 24 * 7).Unix())
	if expiredAt > acc.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return nil, nil, fmt.Errorf("account expiring soon")
	}

	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	listRecord := make([]tables.TableSmtRecordInfo, 0)
	listKeyValue := make([]tables.MintSignInfoKeyValue, 0)
	smtKv := make([]smt.SmtKv, 0)

	for _, v := range req.SubAccountList {
		subAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(v.Account))
		subAcc, err := h.DbDao.GetAccountInfoByAccountId(subAccountId)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeDbError, "failed to query account")
			return nil, nil, fmt.Errorf("GetAccountInfoByAccountId: %s", err.Error())
		}

		tmp := tables.TableSmtRecordInfo{
			SvrName:         config.Cfg.Slb.SvrName,
			AccountId:       subAccountId,
			Nonce:           subAcc.Nonce + 1,
			RecordType:      tables.RecordTypeDefault,
			MintType:        tables.MintTypeManual,
			Action:          common.DasActionUpdateSubAccount,
			ParentAccountId: parentAccountId,
			Account:         v.Account,
			EditKey:         common.EditKeyManual,
			RenewYears:      v.RenewYears,
			Timestamp:       time.Now().UnixNano() / 1e6,
			SubAction:       common.SubActionRenew,
		}
		listRecord = append(listRecord, tmp)

		ownerHex := core.DasAddressHex{
			DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
			AddressHex:     subAcc.Owner,
			ChainType:      subAcc.OwnerChainType,
		}
		ownerArgs, err := h.DasCore.Daf().HexToArgs(ownerHex, ownerHex)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "HexToArgs err")
			return nil, nil, fmt.Errorf("HexToArgs err: %s", err.Error())
		}

		smtKey := smt.AccountIdToSmtH256(subAccountId)
		smtValue, err := blake2b.Blake256(ownerArgs)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt value err")
			return nil, nil, fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
		}
		smtKv = append(smtKv, smt.SmtKv{
			Key:   smtKey,
			Value: smtValue,
		})

		listKeyValue = append(listKeyValue, tables.MintSignInfoKeyValue{
			Key:   subAccountId,
			Value: common.Bytes2Hex(ownerArgs),
		})
	}

	tree := smt.NewSmtSrv(*h.SmtServerUrl, "")
	r, err := tree.UpdateSmt(smtKv, smt.SmtOpt{GetProof: false, GetRoot: true})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt update err")
		return nil, nil, fmt.Errorf("tree.Update err: %s", err.Error())
	}
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt root err")
		return nil, nil, fmt.Errorf("tree.Root err: %s", err.Error())
	}
	keyValueStr, _ := json.Marshal(&listKeyValue)

	renewSignInfo := &tables.TableMintSignInfo{
		SmtRoot:   common.Bytes2Hex(r.Root),
		ExpiredAt: expiredAt,
		Timestamp: uint64(nowTime.UnixNano() / 1e6),
		KeyValue:  string(keyValueStr),
		ChainType: addressHex.ChainType,
		Address:   addressHex.AddressHex,
		SignRole:  signRole,
		SubAction: common.SubActionRenew,
	}
	renewSignInfo.InitMintSignId(parentAccountId)
	for i := range listRecord {
		listRecord[i].MintSignId = renewSignInfo.MintSignId
	}
	return listRecord, renewSignInfo, nil
}
