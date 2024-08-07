package handle

import (
	"context"
	"das_sub_account/config"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	api_code "github.com/dotbitHQ/das-lib/http_api"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"time"
)

type ReqApprovalFulfill struct {
	core.ChainTypeAddress
	Account   string `json:"account" binding:"required"`
	isMainAcc bool
}

type RespApprovalFulfill struct {
	SignInfoList
}

func (h *HttpHandle) ApprovalFulfill(ctx *gin.Context) {
	var (
		funcName               = "ApprovalFulfill"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqApprovalFulfill
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx.Request.Context())

	if err = h.doApprovalFulfill(ctx.Request.Context(), &req, &apiResp); err != nil {
		log.Error("doApprovalEnableDelay err:", err.Error(), funcName, clientIp, remoteAddrIP, ctx.Request.Context())
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doApprovalFulfill(ctx context.Context, req *ReqApprovalFulfill, apiResp *api_code.ApiResp) error {
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
		return h.doApprovalFulfillMainAccount(ctx, req, apiResp)
	}
	return h.doApprovalFulfillSubAccount(req, apiResp)
}

func (h *HttpHandle) doApprovalFulfillMainAccount(ctx context.Context, req *ReqApprovalFulfill, apiResp *api_code.ApiResp) error {
	now := time.Now()
	accInfo, accountBuilder, _, err := h.doApprovalFulfillCheck(req, now, apiResp)
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

	params := common.Hex2Bytes(common.ParamOwner)
	if uint64(now.Unix()) > accountBuilder.AccountApproval.Params.Transfer.SealedUntil {
		params = []byte{}
	}

	// witness action
	actionWitness, err := witness.GenActionDataWitness(common.DasActionFulfillApproval, params)
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
		Action: common.DasActionFulfillApproval,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)
	accData = append(accData, res.Transaction.OutputsData[accountBuilder.Index][common.HashBytesLen:]...)

	contractDas, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: res.Transaction.Outputs[accountBuilder.Index].Capacity,
		Lock:     contractDas.ToScript(accountBuilder.AccountApproval.Params.Transfer.ToLock.Args),
		Type:     contractAcc.ToScript(nil),
	})
	txParams.OutputsData = append(txParams.OutputsData, accData)

	signList, txHash, err := h.buildTx(ctx, &paramBuildTx{
		txParams:  &txParams,
		action:    common.DasActionFulfillApproval,
		account:   req.Account,
		address:   accInfo.Owner,
		chainType: accInfo.OwnerChainType,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}
	log.Info(ctx, "doApprovalEnableAccount: ", txHash)

	if len(signList.List) > 0 {
		signList.List = nil
	}
	resp := RespApprovalEnable{
		SignInfoList: *signList,
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doApprovalFulfillSubAccount(req *ReqApprovalFulfill, apiResp *api_code.ApiResp) error {
	now := time.Now()
	subAcc, _, subOldData, err := h.doApprovalFulfillCheck(req, now, apiResp)
	if err != nil {
		return err
	}
	if apiResp.ErrNo != 0 {
		return nil
	}

	expiredAt := uint64(now.Add(time.Hour * 24 * 7).Unix())
	if expiredAt > subAcc.ExpiredAt {
		apiResp.ApiRespErr(api_code.ApiCodeAccountExpiringSoon, "account expiring soon")
		return fmt.Errorf("account expiring soon")
	}

	approvalInfo, err := h.DbDao.GetAccountPendingApproval(subAcc.AccountId)
	if err != nil {
		return err
	}
	if approvalInfo.ID == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalNotExist, "account approval not exist")
		return fmt.Errorf("account approval not exist")
	}
	ownerHex := core.DasAddressHex{
		DasAlgorithmId: subAcc.OwnerChainType.ToDasAlgorithmId(true),
		AddressHex:     subAcc.Owner,
		ChainType:      subAcc.OwnerChainType,
	}

	var signRole string
	var algId common.DasAlgorithmId
	if uint64(time.Now().Unix()) < approvalInfo.SealedUntil {
		algId = ownerHex.DasAlgorithmId
		signRole = common.ParamOwner
	}

	listRecord := make([]tables.TableSmtRecordInfo, 0)
	listRecord = append(listRecord, tables.TableSmtRecordInfo{
		SvrName:         config.Cfg.Slb.SvrName,
		AccountId:       subAcc.AccountId,
		Nonce:           subAcc.Nonce + 1,
		RecordType:      tables.RecordTypeDefault,
		Action:          common.DasActionUpdateSubAccount,
		SubAction:       common.SubActionFullfillApproval,
		ParentAccountId: subAcc.ParentAccountId,
		Account:         subAcc.Account,
		Timestamp:       now.UnixNano() / 1e6,
		ExpiredAt:       expiredAt,
		SignRole:        signRole,
	})

	// sign info
	dataCache := UpdateSubAccountCache{
		AccountId:     subAcc.AccountId,
		Account:       req.Account,
		Nonce:         subAcc.Nonce + 1,
		ChainType:     ownerHex.ChainType,
		AlgId:         ownerHex.DasAlgorithmId,
		Address:       ownerHex.AddressHex,
		SubAction:     common.SubActionFullfillApproval,
		ListSmtRecord: listRecord,
		ExpiredAt:     expiredAt,
	}

	accApproval, err := subOldData.AccountApproval.GenToMolecule()
	if err != nil {
		return err
	}
	signData := dataCache.GetApprovalSignData(algId, accApproval, apiResp)
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
			SubAction: common.SubActionFullfillApproval,
			SignKey:   signKey,
			SignList: []txbuilder.SignData{
				signData,
			},
		},
	}
	apiResp.ApiRespOK(resp)
	return nil
}

func (h *HttpHandle) doApprovalFulfillCheck(req *ReqApprovalFulfill, now time.Time, apiResp *api_code.ApiResp) (accInfo tables.TableAccountInfo, accountBuilder *witness.AccountCellDataBuilder, oldData *witness.SubAccountData, err error) {
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

	approval, err := h.DbDao.GetAccountPendingApproval(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "Failed to query parent account")
		err = fmt.Errorf("GetAccountApprovalByAccountId err: %s %s", err.Error(), accountId)
		return
	}
	if approval.ID == 0 {
		err = fmt.Errorf("pending approval info not exist: %s", accountId)
		apiResp.ApiRespErr(api_code.ApiCodeAccountApprovalNotExist, err.Error())
		err = fmt.Errorf("GetAccountApprovalByAccountId err: %s %s", err.Error(), accountId)
		return
	}

	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return
	}
	nowUntil := uint64(timeCell.Timestamp())
	if nowUntil < approval.SealedUntil {
		var ownerHex *core.DasAddressHex
		ownerHex, err = req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
			err = fmt.Errorf("FormatChainTypeAddress err: %s", err.Error())
			return
		}
		if !strings.EqualFold(ownerHex.AddressHex, approval.Owner) {
			apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "permission denied")
			return
		}
	}

	var txRes *types.TransactionWithStatus
	if req.isMainAcc {
		accOutPoint := common.String2OutPointStruct(accInfo.Outpoint)
		txRes, err = h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		accountBuilder, err = witness.AccountCellDataBuilderFromTxByName(txRes.Transaction, common.DataTypeNew, req.Account)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
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
		txRes, err = h.DasCore.Client().GetTransaction(h.Ctx, approvalOutpoint.TxHash)
		if err != nil {
			err = fmt.Errorf("GetTransaction err: %s", err.Error())
			apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
			return
		}
		builderMap, err = sanb.SubAccountNewMapFromTx(txRes.Transaction)
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
	}
	return
}
