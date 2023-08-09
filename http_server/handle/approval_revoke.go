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

type ReqApprovalRevoke struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	isMainAcc bool
}

type RespApprovalRevoke struct {
	SignInfoList
}

func (h *HttpHandle) ApprovalRevoke(ctx *gin.Context) {
	var (
		funcName               = "ApprovalRevoke"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqApprovalRevoke
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

	if err = h.doApprovalRevoke(&req, &apiResp); err != nil {
		log.Error("doApprovalEnableDelay err:", err.Error(), funcName, clientIp, remoteAddrIP)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doApprovalRevoke(req *ReqApprovalRevoke, apiResp *api_code.ApiResp) error {
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
		return h.doApprovalRevokeMainAccount(req, apiResp)
	}
	return h.doApprovalRevokeSubAccount(req, apiResp)
}

func (h *HttpHandle) doApprovalRevokeMainAccount(req *ReqApprovalRevoke, apiResp *api_code.ApiResp) error {
	accInfo, accountBuilder, _, err := h.doApprovalRevokeCheck(req, apiResp)
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
	actionWitness, err := witness.GenActionDataWitness(common.DasActionDelayApproval, nil)
	if err != nil {
		return fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// witness account cell
	res, err := h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
	if err != nil {
		return fmt.Errorf("GetTransaction err: %s", err.Error())
	}

	accWitness, accData, err := accountBuilder.GenWitness(&witness.AccountCellParam{
		Action:          common.DasActionRevokeApproval,
		AccountApproval: accountBuilder.AccountApproval,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)
	accData = append(accData, res.Transaction.OutputsData[accountBuilder.Index][common.HashBytesLen:]...)

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
		Capacity: res.Transaction.Outputs[accountBuilder.Index].Capacity,
		Lock:     contractDas.ToScript(lockArgs),
		Type:     contractAcc.ToScript(nil),
	})
	txParams.OutputsData = append(txParams.OutputsData, accData)

	signList, txHash, err := h.buildTx(&paramBuildTx{
		txParams: &txParams,
		action:   common.DasActionRevokeApproval,
		account:  req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}
	log.Info("doApprovalEnableAccount: ", txHash)

	resp := RespApprovalEnable{
		SignInfoList: *signList,
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doApprovalRevokeSubAccount(req *ReqApprovalRevoke, apiResp *api_code.ApiResp) error {
	subAcc, _, subOldData, err := h.doApprovalRevokeCheck(req, apiResp)
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

	listRecord = append(listRecord, tables.TableSmtRecordInfo{
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       subAcc.AccountId,
		Nonce:           subAcc.Nonce + 1,
		RecordType:      tables.RecordTypeDefault,
		Action:          common.DasActionUpdateSubAccount,
		SubAction:       common.SubActionRevokeApproval,
		ParentAccountId: subAcc.ParentAccountId,
		Account:         subAcc.Account,
		Timestamp:       now.UnixNano() / 1e6,
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
		SubAction: common.SubActionRevokeApproval,
	}
	signInfo.InitMintSignId(subAcc.ParentAccountId)
	for i := range listRecord {
		listRecord[i].MintSignId = signInfo.MintSignId
	}

	// sign info
	dataCache := UpdateSubAccountCache{
		AccountId:     subAcc.AccountId,
		Account:       req.Account,
		Nonce:         subAcc.Nonce + 1,
		ChainType:     ownerHex.ChainType,
		AlgId:         ownerHex.DasAlgorithmId,
		Address:       ownerHex.AddressHex,
		SubAction:     common.SubActionRevokeApproval,
		MinSignInfo:   signInfo,
		ListSmtRecord: listRecord,
	}

	accApproval, err := subOldData.AccountApproval.GenToMolecule()
	if err != nil {
		return err
	}
	approvalInfo, err := h.DbDao.GetAccountApprovalByAccountId(subAcc.AccountId)
	if err != nil {
		return err
	}
	if approvalInfo.ID == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalNotExist, "account approval not exist")
		return fmt.Errorf("account approval not exist")
	}
	if uint64(time.Now().Unix()) < approvalInfo.ProtectedUntil {
		apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalProtected, "account approval protected")
		return fmt.Errorf("account approval protected")
	}

	signData := dataCache.GetApprovalSignData(common.DasAlgorithmIdEth, accApproval, apiResp)
	if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	dataCache.OldSignMsg = signData.SignMsg
	dataCache.SignData = signData

	// cache
	signKey := dataCache.CacheKey()
	cacheStr := toolib.JsonString(&dataCache)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	resp := RespApprovalEnable{
		SignInfoList: SignInfoList{
			Action:    common.DasActionUpdateSubAccount,
			SubAction: common.SubActionRevokeApproval,
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

func (h *HttpHandle) doApprovalRevokeCheck(req *ReqApprovalRevoke, apiResp *api_code.ApiResp) (accInfo *tables.TableAccountInfo, builder *witness.AccountCellDataBuilder, data *witness.SubAccountData, err error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		err = fmt.Errorf("GetAccountInfoByAccountId err: %s %s", err.Error(), accountId)
		return
	}
	if accInfo.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeParentAccountNotExist, "parent account does not exist")
		return
	}
	if accInfo.Status != tables.AccountStatusApproval {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not approval")
		return
	}

	platformHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, false)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}
	approval, err := h.DbDao.GetAccountApprovalByAccountIdAndPlatform(accountId, platformHex.AddressHex)
	if err != nil || approval.ID == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query approval")
		err = fmt.Errorf("GetAccountApprovalByAccountIdAndPlatform err: %s %s %s", err.Error(), accountId, platformHex.AddressHex)
		return
	}

	var accountBuilder *witness.AccountCellDataBuilder
	var oldData *witness.SubAccountData
	if req.isMainAcc {
		accOutPoint := common.String2OutPointStruct(accInfo.Outpoint)
		res, err := h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return nil, nil, nil, fmt.Errorf("GetTransaction err: %s", err.Error())
		}
		accountBuilder, err = witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("AccountCellDataBuilderMapFromTx err: %s", err.Error())
		}

		if uint64(time.Now().Unix()) <= accountBuilder.AccountApproval.Params.Transfer.ProtectedUntil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "protected_until invalid")
			return nil, nil, nil, fmt.Errorf("protected_until invalid")
		}
	} else {
		_, subAccountBuilderMap, err := h.TxTool.GetOldSubAccount([]string{accountId}, "")
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return nil, nil, nil, fmt.Errorf("GetOldSubAccount err: %s", err.Error())
		}
		subAccountData := subAccountBuilderMap[accountId]
		oldData = subAccountData.CurrentSubAccountData

		if uint64(time.Now().Unix()) <= oldData.AccountApproval.Params.Transfer.ProtectedUntil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "protected_until invalid")
			return nil, nil, nil, fmt.Errorf("protected_until invalid")
		}
	}
	return &acc, accountBuilder, oldData, nil
}
