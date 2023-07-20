package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
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

type ReqApprovalEnable struct {
	Platform       core.ChainTypeAddress `json:"platform" binding:"required"`
	Owner          core.ChainTypeAddress `json:"owner" binding:"required"`
	To             core.ChainTypeAddress `json:"to" binding:"required"`
	Account        string                `json:"account" binding:"required"`
	ProtectedUntil uint64                `json:"protected_until" binding:"required"`
	SealedUntil    uint64                `json:"sealed_until" binding:"required"`
	isMainAcc      bool
}

type RespApprovalEnable struct {
	SignInfoList
}

func (h *HttpHandle) ApprovalEnable(ctx *gin.Context) {
	var (
		funcName               = "ReqApprovalEnable"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqApprovalEnable
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req))

	if err = h.doApprovalEnableEnable(&req, &apiResp); err != nil {
		log.Error("ApprovalEnable err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doApprovalEnableEnable(req *ReqApprovalEnable, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	accountSection := strings.Split(req.Account, ".")
	if len(accountSection) != 2 && len(accountSection) != 3 {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "account invalid")
		return nil
	}
	req.isMainAcc = len(accountSection) == 2

	if req.isMainAcc {
		return h.doApprovalEnableMainAccount(req, apiResp)
	}
	return h.doApprovalEnableSubAccount(req, apiResp)
}

func (h *HttpHandle) doApprovalEnableMainAccount(req *ReqApprovalEnable, apiResp *api_code.ApiResp) error {
	accInfo, platformLockBs, toLockBs, err := h.doApprovalEnableCheck(req, apiResp)
	if err != nil {
		return err
	}
	if apiResp.ErrNo != 0 {
		return nil
	}

	var txParams txbuilder.BuildTransactionParams

	contractAcc, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		return fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	heightCell, err := h.DasCore.GetHeightCell()
	if err != nil {
		return fmt.Errorf("GetHeightCell err: %s", err.Error())
	}
	configCellMain, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsMain)
	if err != nil {
		return fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	configCellAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsAccount)
	if err != nil {
		return fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	txParams.CellDeps = append(txParams.CellDeps,
		contractAcc.ToCellDep(),
		timeCell.ToCellDep(),
		heightCell.ToCellDep(),
		configCellMain.ToCellDep(),
		configCellAcc.ToCellDep())

	// inputs account cell
	accOutPoint := common.String2OutPointStruct(accInfo.Outpoint)
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: accOutPoint,
	})

	// witness action
	actionWitness, err := witness.GenActionDataWitness(common.DasActionCreateApproval, []byte(common.ParamOwner))
	if err != nil {
		return fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// witness account cell
	res, err := h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
	if err != nil {
		return fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	builderMap, err := witness.AccountCellDataBuilderMapFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		return fmt.Errorf("AccountCellDataBuilderMapFromTx err: %s", err.Error())
	}
	builder, ok := builderMap[req.Account]
	if !ok {
		return fmt.Errorf("builderMap not exist account: %s", req.Account)
	}

	accWitness, accData, err := builder.GenWitness(&witness.AccountCellParam{
		Action: common.DasActionCreateApproval,
		Status: common.AccountStatusOnApproval,
		AccountApproval: witness.AccountApproval{
			Action: witness.AccountApprovalActionTransfer,
			Params: witness.AccountApprovalTransferParams{
				PlatformLock:     platformLockBs,
				ProtectedUntil:   req.ProtectedUntil,
				SealedUntil:      req.SealedUntil,
				DelayCountRemain: 1,
				ToLock:           toLockBs,
			},
		},
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)
	accData = append(accData, res.Transaction.OutputsData[builder.Index][common.HashBytesLen:]...)

	capacity := res.Transaction.Outputs[builder.Index].Capacity

	lockArgs, err := h.DasCore.Daf().HexToArgs(core.DasAddressHex{
		DasAlgorithmId: accInfo.OwnerChainType.ToDasAlgorithmId(true),
		AddressHex:     accInfo.Owner,
		ChainType:      accInfo.OwnerChainType,
	}, core.DasAddressHex{
		DasAlgorithmId: accInfo.ManagerChainType.ToDasAlgorithmId(true),
		AddressHex:     accInfo.Manager,
		ChainType:      accInfo.ManagerChainType,
	})
	if err != nil {
		return fmt.Errorf("HexToArgs err: %s", err.Error())
	}

	contractDas, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}

	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: capacity,
		Lock:     contractDas.ToScript(lockArgs),
		Type:     contractAcc.ToScript(nil),
	})
	txParams.OutputsData = append(txParams.OutputsData, accData)

	signKey, signList, txHash, err := h.buildTx(&paramBuildTx{
		txParams: &txParams,
		action:   common.DasActionCreateApproval,
		account:  req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}
	log.Info("doApprovalEnableAccount: ", txHash)

	resp := RespApprovalEnable{
		SignInfoList: SignInfoList{
			Action:  common.DasActionCreateApproval,
			SignKey: signKey,
			List: []SignInfo{
				{
					signList,
				},
			},
		},
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doApprovalEnableSubAccount(req *ReqApprovalEnable, apiResp *api_code.ApiResp) error {
	subAcc, _, _, err := h.doApprovalEnableCheck(req, apiResp)
	if err != nil {
		return err
	}
	if apiResp.ErrNo != 0 {
		return nil
	}

	now := time.Now()
	expiredAt := uint64(now.Add(time.Hour * 24 * 7).Unix())
	if expiredAt > subAcc.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return fmt.Errorf("account expiring soon")
	}

	listRecord := make([]tables.TableSmtRecordInfo, 0)
	listKeyValue := make([]tables.MintSignInfoKeyValue, 0)
	smtKv := make([]smt.SmtKv, 0)

	reqData, _ := json.Marshal(req)

	listRecord = append(listRecord, tables.TableSmtRecordInfo{
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       subAcc.AccountId,
		Nonce:           subAcc.Nonce + 1,
		RecordType:      tables.RecordTypeDefault,
		Action:          common.DasActionUpdateSubAccount,
		SubAction:       common.DasActionCreateApproval,
		ParentAccountId: subAcc.ParentAccountId,
		Account:         subAcc.Account,
		EditKey:         common.EditKeyApproval,
		Timestamp:       now.UnixNano() / 1e6,
		Content:         string(reqData),
	})

	ownerHex := core.DasAddressHex{
		DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
		AddressHex:     subAcc.Owner,
		ChainType:      subAcc.OwnerChainType,
	}
	ownerArgs, err := h.DasCore.Daf().HexToArgs(ownerHex, ownerHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "HexToArgs err")
		return fmt.Errorf("HexToArgs err: %s", err.Error())
	}

	smtKey := smt.AccountIdToSmtH256(subAcc.AccountId)
	smtValue, err := blake2b.Blake256(ownerArgs)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt value err")
		return fmt.Errorf("blake2b.Blake256 err: %s", err.Error())
	}
	smtKv = append(smtKv, smt.SmtKv{
		Key:   smtKey,
		Value: smtValue,
	})

	listKeyValue = append(listKeyValue, tables.MintSignInfoKeyValue{
		Key:   subAcc.AccountId,
		Value: common.Bytes2Hex(ownerArgs),
	})

	tree := smt.NewSmtSrv(*h.SmtServerUrl, "")
	r, err := tree.UpdateSmt(smtKv, smt.SmtOpt{GetProof: false, GetRoot: true})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt update err")
		return fmt.Errorf("tree.Update err: %s", err.Error())
	}
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "smt root err")
		return fmt.Errorf("tree.Root err: %s", err.Error())
	}
	keyValueStr, _ := json.Marshal(&listKeyValue)

	signInfo := &tables.TableMintSignInfo{
		SmtRoot:   common.Bytes2Hex(r.Root),
		ExpiredAt: expiredAt,
		Timestamp: uint64(now.UnixNano() / 1e6),
		KeyValue:  string(keyValueStr),
		ChainType: ownerHex.ChainType,
		Address:   ownerHex.AddressHex,
		SignRole:  common.ParamOwner,
		SubAction: common.DasActionCreateApproval,
	}
	signInfo.InitMintSignId(subAcc.ParentAccountId)
	for i := range listRecord {
		listRecord[i].MintSignId = signInfo.MintSignId
	}

	// sign info
	dataCache := UpdateSubAccountCache{
		ParentAccountId: subAcc.ParentAccountId,
		Account:         req.Account,
		ChainType:       ownerHex.ChainType,
		AlgId:           ownerHex.DasAlgorithmId,
		Address:         ownerHex.AddressHex,
		SubAction:       common.DasActionCreateApproval,
		MinSignInfo:     signInfo,
		ListSmtRecord:   listRecord,
	}
	signData := dataCache.GetCreateSignData(ownerHex.DasAlgorithmId, apiResp)
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

	resp := RespApprovalEnable{
		SignInfoList: SignInfoList{
			Action:    common.DasActionUpdateSubAccount,
			SubAction: common.DasActionCreateApproval,
			SignKey:   signKey,
			List: []SignInfo{
				{
					SignList: []txbuilder.SignData{
						signData,
					},
				},
			},
		},
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doApprovalEnableCheck(req *ReqApprovalEnable, apiResp *api_code.ApiResp) (accInfo tables.TableAccountInfo, platformLockBs []byte, toLockBs []byte, err error) {
	nowTime := time.Now()
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err = h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		err = fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), accountId)
		return
	} else if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
		return
	} else if accInfo.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not normal")
		return
	} else if accInfo.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "parent account is expired")
		return
	}
	if accInfo.ExpiredAt-uint64(nowTime.Unix()) < 3600*24*30 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return
	}

	keys := []core.ChainTypeAddress{req.Platform, req.Owner, req.To}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i].KeyInfo.CoinType == keys[j].KeyInfo.CoinType &&
				strings.EqualFold(keys[i].KeyInfo.Key, keys[j].KeyInfo.Key) {
				apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "address is repeated")
				return
			}
		}
	}

	ownerHexAddress, err := req.Owner.FormatChainTypeAddress(config.Cfg.Server.Net, false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "owner address invalid")
		err = fmt.Errorf("FormatChainTypeAddress err:%s", err.Error())
		return
	}
	if accInfo.OwnerChainType != ownerHexAddress.ChainType ||
		!strings.EqualFold(accInfo.Owner, ownerHexAddress.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return
	}

	var platformLock *types.Script
	platformLock, _, err = req.Platform.FormatChainTypeAddressToScript(config.Cfg.Server.Net, false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "owner address invalid")
		err = fmt.Errorf("FormatChainTypeAddress err:%s", err.Error())
		return
	}
	platformLockBs, _ = platformLock.Serialize()

	var toLock *types.Script
	toLock, _, err = req.To.FormatChainTypeAddressToScript(config.Cfg.Server.Net, false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "owner address invalid")
		err = fmt.Errorf("FormatChainTypeAddress err:%s", err.Error())
		return
	}
	toLockBs, _ = toLock.Serialize()
	return
}
