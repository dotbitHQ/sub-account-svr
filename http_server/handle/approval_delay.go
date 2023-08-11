package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"errors"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqApprovalDelay struct {
	core.ChainTypeAddress
	Account     string `json:"account" binding:"required"`
	SealedUntil uint64 `json:"sealed_until" binding:"required"`
	EvmChainId  int64  `json:"evm_chain_id"`
	isMainAcc   bool
}

type RespApprovalDelay struct {
	SignInfoList
}

func (h *HttpHandle) ApprovalDelay(ctx *gin.Context) {
	var (
		funcName               = "ReqApprovalDelay"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqApprovalDelay
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

	if err = h.doApprovalDelay(&req, &apiResp); err != nil {
		log.Error("doApprovalEnableDelay err:", err.Error(), funcName, clientIp, remoteAddrIP)
		if apiResp.ErrNo == 0 {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		}
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doApprovalDelay(req *ReqApprovalDelay, apiResp *api_code.ApiResp) error {
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
		return h.doApprovalDelayMainAccount(req, apiResp)
	}
	return h.doApprovalDelaySubAccount(req, apiResp)
}

func (h *HttpHandle) doApprovalDelayMainAccount(req *ReqApprovalDelay, apiResp *api_code.ApiResp) error {
	accInfo, accountBuilder, _, err := h.doApprovalDelayCheck(req, apiResp)
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
	accWitness, accData, err := accountBuilder.GenWitness(&witness.AccountCellParam{
		Action: common.DasActionDelayApproval,
		AccountApproval: witness.AccountApproval{
			Action: witness.AccountApprovalActionTransfer,
			Params: witness.AccountApprovalParams{
				Transfer: witness.AccountApprovalParamsTransfer{
					SealedUntil: req.SealedUntil,
				},
			},
		},
	})
	if err != nil {
		log.Error("GenWitness err:", err.Error())
		return err
	}
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	res, err := h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
	if err != nil {
		return fmt.Errorf("GetTransaction err: %s", err.Error())
	}
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
		txParams:   &txParams,
		action:     common.DasActionDelayApproval,
		account:    req.Account,
		evmChainId: req.EvmChainId,
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

func (h *HttpHandle) doApprovalDelaySubAccount(req *ReqApprovalDelay, apiResp *api_code.ApiResp) error {
	subAcc, _, oldData, err := h.doApprovalDelayCheck(req, apiResp)
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

	oldData.AccountApproval.Params.Transfer.SealedUntil = req.SealedUntil
	approvalMol, err := oldData.AccountApproval.GenToMolecule()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return err
	}

	listRecord := make([]tables.TableSmtRecordInfo, 0)
	listRecord = append(listRecord, tables.TableSmtRecordInfo{
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       subAcc.AccountId,
		Nonce:           subAcc.Nonce + 1,
		RecordType:      tables.RecordTypeDefault,
		Action:          common.DasActionUpdateSubAccount,
		SubAction:       common.DasActionDelayApproval,
		ParentAccountId: subAcc.ParentAccountId,
		Account:         subAcc.Account,
		EditKey:         common.EditKeyApproval,
		EditValue:       common.Bytes2Hex(approvalMol.AsSlice()),
		Timestamp:       now.UnixNano() / 1e6,
		ExpiredAt:       expiredAt,
		SignRole:        common.ParamOwner,
	})

	ownerHex := core.DasAddressHex{
		DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
		AddressHex:     subAcc.Owner,
		ChainType:      subAcc.OwnerChainType,
	}

	// sign info
	dataCache := UpdateSubAccountCache{
		AccountId:     subAcc.AccountId,
		Account:       req.Account,
		Nonce:         subAcc.Nonce + 1,
		ChainType:     ownerHex.ChainType,
		AlgId:         ownerHex.DasAlgorithmId,
		Address:       ownerHex.AddressHex,
		SubAction:     common.DasActionDelayApproval,
		ListSmtRecord: listRecord,
		ExpiredAt:     expiredAt,
	}
	signData := dataCache.GetApprovalSignData(ownerHex.DasAlgorithmId, approvalMol, apiResp)
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
			SubAction: common.DasActionDelayApproval,
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

func (h *HttpHandle) doApprovalDelayCheck(req *ReqApprovalDelay, apiResp *api_code.ApiResp) (accInfo tables.TableAccountInfo, builder *witness.AccountCellDataBuilder, oldData *witness.SubAccountData, err error) {
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	accInfo, err = h.DbDao.GetAccountInfoByAccountId(accountId)
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

	ownerHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return
	}
	if accInfo.OwnerChainType != ownerHex.ChainType || !strings.EqualFold(accInfo.Owner, ownerHex.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
		return
	}

	var res *types.TransactionWithStatus
	var accountBuilder *witness.AccountCellDataBuilder
	if req.isMainAcc {
		accOutPoint := common.String2OutPointStruct(accInfo.Outpoint)
		res, err = h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
		if err != nil {
			err = fmt.Errorf("GetTransaction err: %s", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		accountBuilder, err = witness.AccountCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
		if err != nil {
			err = fmt.Errorf("AccountCellDataBuilderMapFromTx err: %s", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		if req.SealedUntil <= accountBuilder.AccountApproval.Params.Transfer.SealedUntil {
			err = errors.New("sealed_until invalid")
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
			return
		}
	} else {
		var approvalInfo tables.ApprovalInfo
		var sanb witness.SubAccountNewBuilder
		var builderMap map[string]*witness.SubAccountNew

		approvalInfo, err = h.DbDao.GetAccountPendingApproval(accountId)
		if err != nil {
			return
		}
		if approvalInfo.ID == 0 {
			err = fmt.Errorf("pending approval info not exist: %s", accountId)
			apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalNotExist, err.Error())
			return
		}
		approvalOutpoint := common.String2OutPointStruct(approvalInfo.Outpoint)
		res, err = h.DasCore.Client().GetTransaction(h.Ctx, approvalOutpoint.TxHash)
		if err != nil {
			err = fmt.Errorf("GetTransaction err: %s", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		builderMap, err = sanb.SubAccountNewMapFromTx(res.Transaction)
		if err != nil {
			return
		}
		subAccountData, ok := builderMap[accountId]
		if !ok {
			err = fmt.Errorf("accountId no exist in tx: %s", accountId)
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		oldData = subAccountData.CurrentSubAccountData

		if req.SealedUntil <= oldData.AccountApproval.Params.Transfer.SealedUntil {
			err = fmt.Errorf("request sealed_until: %d can not less than old sealed_until: %d", req.SealedUntil, oldData.AccountApproval.Params.Transfer.SealedUntil)
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
			return
		}
	}
	return
}
