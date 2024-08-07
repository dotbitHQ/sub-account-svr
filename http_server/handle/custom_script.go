package handle

import (
	"bytes"
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
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqCustomScript struct {
	core.ChainTypeAddress
	Account            string                              `json:"account"`
	CustomScriptArgs   string                              `json:"custom_script_args"`
	CustomScriptConfig map[uint8]witness.CustomScriptPrice `json:"custom_script_config"`
}

type RespCustomScript struct {
	SignInfoList
}

func (h *HttpHandle) CustomScript(ctx *gin.Context) {
	var (
		funcName               = "CustomScript"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqCustomScript
		apiResp                api_code.ApiResp
		err                    error
	)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Error("ShouldBindJSON err: ", err.Error(), funcName, clientIp, ctx)
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		ctx.JSON(http.StatusOK, apiResp)
		return
	}
	log.Info("ApiReq:", funcName, clientIp, remoteAddrIP, toolib.JsonString(req), ctx)

	if err = h.doCustomScript(ctx, &req, &apiResp); err != nil {
		log.Error("doCustomScript err:", err.Error(), funcName, clientIp, ctx)
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCustomScript(ctx context.Context, req *ReqCustomScript, apiResp *api_code.ApiResp) error {
	var resp RespCustomScript

	hexAddress, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return nil
	}
	if req.Account == "" {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "account is nil")
		return nil
	}
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	// check account
	accountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))
	acc, err := h.DbDao.GetAccountInfoByAccountId(accountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeDbError, "GetAccountInfoByAccountId err")
		return fmt.Errorf("GetAccountInfoByAccountId err: %s", err.Error())
	} else if acc.Id == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeAccountNotExist, "account not exist")
		return nil
	} else if acc.Status != tables.AccountStatusNormal {
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account status is not normal")
		return nil
	} else if acc.IsExpired() {
		apiResp.ApiRespErr(api_code.ApiCodeAccountIsExpired, "account is expired")
		return nil
	} else if acc.OwnerChainType != hexAddress.ChainType || !strings.EqualFold(acc.Owner, hexAddress.AddressHex) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "owner permission required")
		return nil
	} else if acc.EnableSubAccount != tables.AccountEnableStatusOn {
		apiResp.ApiRespErr(api_code.ApiCodeEnableSubAccountIsOff, "sub-account not enabled")
		return nil
	}

	// build tx
	customScriptArgs := make([]byte, 32)
	if req.CustomScriptArgs != "" {
		tmpArgs := common.Hex2Bytes(req.CustomScriptArgs)
		if len(tmpArgs) != 32 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "CustomScriptArgs err")
			return nil
		}
		customScriptArgs = tmpArgs
	}
	contractSubAcc, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "GetDasContractInfo err")
		return fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	subAccountLiveCell, err := h.DasCore.GetSubAccountCell(acc.AccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "GetSubAccountCell err")
		return fmt.Errorf("getSubAccountCell err: %s", err.Error())
	}
	subDataDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	_, hashConfig := witness.BuildCustomScriptConfig(witness.CustomScriptConfig{
		Header:    witness.Script001,
		Version:   0,
		Body:      req.CustomScriptConfig,
		MaxLength: 0,
	})
	if bytes.Compare(subDataDetail.CustomScriptArgs, customScriptArgs) == 0 && bytes.Compare(subDataDetail.CustomScriptConfig, hashConfig) == 0 {
		apiResp.ApiRespErr(api_code.ApiCodeSameCustomScript, "same custom script")
		return nil
	}
	// check custom script
	var defaultCustomScriptArgs = make([]byte, 32)
	if bytes.Compare(customScriptArgs, defaultCustomScriptArgs) != 0 {
		subDataDetail.Flag = witness.FlagTypeCustomPrice
		subDataDetail.CustomScriptArgs = customScriptArgs
		subDataDetail.CustomScriptConfig = hashConfig
		subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
		_, err = h.DasCore.GetCustomScriptLiveCell(subAccountOutputData)
		if err != nil {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "custom-script-args invalid")
			return fmt.Errorf("GetCustomScriptLiveCell err: %s", err.Error())
		}
	}

	p := paramCustomScriptTx{
		acc:                &acc,
		customScriptArgs:   customScriptArgs,
		subAccountLiveCell: subAccountLiveCell,
		contractSubAcc:     contractSubAcc,
		customScriptConfig: req.CustomScriptConfig,
	}
	txParams, err := h.buildCustomScriptTx(&p)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildCustomScriptTx err")
		return fmt.Errorf("buildCustomScriptTx err: %s", err.Error())
	}

	signList, _, err := h.buildTx(ctx, &paramBuildTx{
		txParams:   txParams,
		skipGroups: []int{1},
		chainType:  hexAddress.ChainType,
		address:    hexAddress.AddressHex,
		action:     common.DasActionConfigSubAccountCustomScript,
		account:    req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err")
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp.Action = common.DasActionConfigSubAccountCustomScript
	resp.SignKey = signList.SignKey
	resp.List = signList.List
	resp.SignList = signList.List[0].SignList
	log.Info(ctx, "doCustomScript:", toolib.JsonString(resp))
	apiResp.ApiRespOK(resp)
	return nil
}

type paramBuildTx struct {
	txParams   *txbuilder.BuildTransactionParams
	skipGroups []int
	chainType  common.ChainType
	address    string
	action     common.DasAction
	subAction  common.SubAction
	account    string
	evmChainId int64
}

func (h *HttpHandle) buildTx(ctx context.Context, p *paramBuildTx) (*SignInfoList, string, error) {
	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(p.txParams); err != nil {
		return nil, "", fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	switch p.action {
	case common.DasActionConfigSubAccountCustomScript:
		sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
		changeCapacity := txBuilder.Transaction.Outputs[1].Capacity - sizeInBlock - 1000
		txBuilder.Transaction.Outputs[1].Capacity = changeCapacity
	case common.DasActionConfigSubAccount:
		sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
		latestOutput := len(txBuilder.Transaction.Outputs) - 1
		changeCapacity := txBuilder.Transaction.Outputs[latestOutput].Capacity - sizeInBlock - 5000
		txBuilder.Transaction.Outputs[latestOutput].Capacity = changeCapacity
		log.Infof("total size: %d, changeCapacity: %d", sizeInBlock, changeCapacity, ctx)
	case common.DasActionCreateApproval, common.DasActionDelayApproval,
		common.DasActionRevokeApproval, common.DasActionFulfillApproval:
		sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
		changeCapacity := txBuilder.Transaction.Outputs[0].Capacity - sizeInBlock - 1000
		txBuilder.Transaction.Outputs[0].Capacity = changeCapacity
		log.Info(ctx, "buildTx:", p.action, sizeInBlock, changeCapacity)
	}

	signList, err := txBuilder.GenerateDigestListFromTx(p.skipGroups)
	if err != nil {
		return nil, "", fmt.Errorf("GenerateDigestListFromTx err: %s", err.Error())
	}

	var mmJsonObj *common.MMJsonObj
	if signList[0].SignType == common.DasAlgorithmIdEth712 {
		mmJsonObj, err = txBuilder.BuildMMJsonObj(p.evmChainId)
		if err != nil {
			return nil, "", err
		}
	}

	log.Info(ctx, "buildTx:", txBuilder.TxString())

	// cache
	sic := SignInfoCache{
		ChainType: p.chainType,
		Address:   p.address,
		Action:    p.action,
		Account:   p.account,
		BuilderTx: txBuilder.DasTxBuilderTransaction,
	}
	signKey := sic.SignKey()
	cacheStr := toolib.JsonString(&sic)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return nil, "", fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	txHash, err := txBuilder.Transaction.ComputeHash()
	if err != nil {
		return nil, "", err
	}

	signListInfo := SignInfoList{
		Action:    p.action,
		SubAction: p.subAction,
		SignKey:   signKey,
	}
	if mmJsonObj != nil {
		signListInfo.SignList = signList
		signListInfo.MMJson = mmJsonObj
		signListInfo.Is712 = true
	} else {
		signListInfo.List = append(signListInfo.List, SignInfo{signList})
		signListInfo.SignList = signList
	}
	return &signListInfo, txHash.Hex(), nil
}

type paramCustomScriptTx struct {
	acc                *tables.TableAccountInfo
	customScriptArgs   []byte
	subAccountLiveCell *indexer.LiveCell
	contractSubAcc     *core.DasContractInfo
	customScriptConfig map[uint8]witness.CustomScriptPrice
}

func (h *HttpHandle) buildCustomScriptTx(p *paramCustomScriptTx) (*txbuilder.BuildTransactionParams, error) {
	var txParams txbuilder.BuildTransactionParams

	// inputs
	accOutPoint := common.String2OutPointStruct(p.acc.Outpoint)
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: accOutPoint,
	})

	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: p.subAccountLiveCell.OutPoint,
	})

	// outputs account cell
	txAcc, err := h.DasCore.Client().GetTransaction(h.Ctx, accOutPoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: txAcc.Transaction.Outputs[accOutPoint.Index].Capacity,
		Lock:     txAcc.Transaction.Outputs[accOutPoint.Index].Lock,
		Type:     txAcc.Transaction.Outputs[accOutPoint.Index].Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, txAcc.Transaction.OutputsData[accOutPoint.Index])

	// custom script witness
	witConfig, hashConfig := witness.BuildCustomScriptConfig(witness.CustomScriptConfig{
		Header:    witness.Script001,
		Version:   0,
		Body:      p.customScriptConfig,
		MaxLength: 0,
	})
	txParams.OtherWitnesses = append(txParams.OtherWitnesses, witConfig)

	// outputs sub-sccount cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: p.subAccountLiveCell.Output.Capacity,
		Lock:     p.subAccountLiveCell.Output.Lock,
		Type:     p.subAccountLiveCell.Output.Type,
	})
	subDataDetail := witness.ConvertSubAccountCellOutputData(p.subAccountLiveCell.OutputData)
	subDataDetail.CustomScriptArgs = p.customScriptArgs
	subDataDetail.CustomScriptConfig = hashConfig
	subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, subAccountOutputData)

	// action witness
	actionWitness, err := witness.GenActionDataWitnessV2(common.DasActionConfigSubAccountCustomScript, nil, common.ParamOwner)
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	// account witness
	builderMap, err := witness.AccountIdCellDataBuilderFromTx(txAcc.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	builder, ok := builderMap[p.acc.AccountId]
	if !ok {
		return nil, fmt.Errorf("builderMap not exist account: %s", p.acc.Account)
	}
	accWitness, _, err := builder.GenWitness(&witness.AccountCellParam{
		OldIndex: 0,
		NewIndex: 0,
		Action:   common.DasActionConfigSubAccountCustomScript,
	})
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// cell deps
	heightCell, err := h.DasCore.GetHeightCell()
	if err != nil {
		return nil, fmt.Errorf("GetHeightCell err: %s", err.Error())
	}
	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		return nil, fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	configCellAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	contractAcc, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractDasLock, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	txParams.CellDeps = append(txParams.CellDeps,
		contractDasLock.ToCellDep(),
		contractAcc.ToCellDep(),
		p.contractSubAcc.ToCellDep(),
		heightCell.ToCellDep(),
		timeCell.ToCellDep(),
		configCellAcc.ToCellDep(),
		configCellSubAcc.ToCellDep(),
	)

	return &txParams, nil
}
