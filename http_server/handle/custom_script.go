package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
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
	Account          string `json:"account"`
	CustomScriptArgs string `json:"custom_script_args"`
}

type RespCustomScript struct {
	SignInfoList
}

func (h *HttpHandle) CustomScript(ctx *gin.Context) {
	var (
		funcName = "CustomScript"
		clientIp = GetClientIp(ctx)
		req      ReqCustomScript
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

	if err = h.doCustomScript(&req, &apiResp); err != nil {
		log.Error("doCustomScript err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doCustomScript(req *ReqCustomScript, apiResp *api_code.ApiResp) error {
	var resp RespCustomScript

	hexAddress, err := req.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, err.Error())
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
		apiResp.ApiRespErr(api_code.ApiCodeAccountStatusOnSaleOrAuction, "account on sale or auction")
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
	customScriptArgs := make([]byte, 33)
	if req.CustomScriptArgs != "" {
		tmpArgs := common.Hex2Bytes(req.CustomScriptArgs)
		if len(tmpArgs) != 33 {
			apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "CustomScriptArgs err")
			return nil
		}
		customScriptArgs = tmpArgs
	}
	p := paramCustomScriptTx{
		acc:              &acc,
		customScriptArgs: customScriptArgs,
	}
	txParams, err := h.buildCustomScriptTx(&p)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildCustomScriptTx err: "+err.Error())
		return fmt.Errorf("buildCustomScriptTx err: %s", err.Error())
	}

	signKey, signList, err := h.buildTx(&paramBuildTx{
		txParams:   txParams,
		skipGroups: []int{1},
		chainType:  hexAddress.ChainType,
		address:    hexAddress.AddressHex,
		action:     common.DasActionConfigSubAccountCustomScript,
		account:    req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp.Action = common.DasActionConfigSubAccountCustomScript
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		//SignKey:  "",
		SignList: signList,
	})

	apiResp.ApiRespOK(resp)
	return nil
}

type paramBuildTx struct {
	txParams   *txbuilder.BuildTransactionParams
	skipGroups []int
	chainType  common.ChainType
	address    string
	action     common.DasAction
	account    string
}

func (h *HttpHandle) buildTx(p *paramBuildTx) (string, []txbuilder.SignData, error) {
	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(p.txParams); err != nil {
		return "", nil, fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	if p.action == common.DasActionConfigSubAccountCustomScript {
		sizeInBlock, _ := txBuilder.Transaction.SizeInBlock()
		changeCapacity := txBuilder.Transaction.Outputs[1].Capacity - sizeInBlock - 1000
		txBuilder.Transaction.Outputs[1].Capacity = changeCapacity
	}

	signList, err := txBuilder.GenerateDigestListFromTx(p.skipGroups)
	if err != nil {
		return "", nil, fmt.Errorf("GenerateDigestListFromTx err: %s", err.Error())
	}

	log.Info("buildTx:", txBuilder.TxString())

	// cache
	sic := SignInfoCache{
		ChainType: p.chainType,
		Address:   p.address,
		Action:    p.action,
		Account:   p.account,
		Capacity:  0,
		BuilderTx: nil,
	}
	sic.BuilderTx = txBuilder.DasTxBuilderTransaction
	signKey := sic.SignKey()
	cacheStr := toolib.JsonString(&sic)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		return "", nil, fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	return signKey, signList, nil
}

type paramCustomScriptTx struct {
	acc              *tables.TableAccountInfo
	customScriptArgs []byte
}

func (h *HttpHandle) buildCustomScriptTx(p *paramCustomScriptTx) (*txbuilder.BuildTransactionParams, error) {
	var txParams txbuilder.BuildTransactionParams

	// inputs
	accOutPoint := common.String2OutPointStruct(p.acc.Outpoint)
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: accOutPoint,
	})

	contractSubAcc, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	subAccountLiveCell, err := h.getSubAccountCell(contractSubAcc, p.acc.AccountId)
	if err != nil {
		return nil, fmt.Errorf("getSubAccountCell err: %s", err.Error())
	}
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: subAccountLiveCell.OutPoint,
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

	// outputs sub-sccount cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: subAccountLiveCell.Output.Capacity,
		Lock:     subAccountLiveCell.Output.Lock,
		Type:     subAccountLiveCell.Output.Type,
	})
	subDataDetail := witness.ConvertSubAccountCellOutputData(subAccountLiveCell.OutputData)
	subDataDetail.CustomScriptArgs = p.customScriptArgs
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
		contractSubAcc.ToCellDep(),
		heightCell.ToCellDep(),
		timeCell.ToCellDep(),
		configCellAcc.ToCellDep(),
		configCellSubAcc.ToCellDep(),
	)

	return &txParams, nil
}

func (h *HttpHandle) getSubAccountCell(contractSubAcc *core.DasContractInfo, parentAccountId string) (*indexer.LiveCell, error) {
	searchKey := indexer.SearchKey{
		Script:     contractSubAcc.ToScript(common.Hex2Bytes(parentAccountId)),
		ScriptType: indexer.ScriptTypeType,
		ArgsLen:    0,
		Filter:     nil,
	}
	subAccLiveCells, err := h.DasCore.Client().GetCells(h.Ctx, &searchKey, indexer.SearchOrderDesc, 1, "")
	if err != nil {
		return nil, fmt.Errorf("GetCells err: %s", err.Error())
	}
	if subLen := len(subAccLiveCells.Objects); subLen != 1 {
		return nil, fmt.Errorf("sub account outpoint len: %d", subLen)
	}
	return subAccLiveCells.Objects[0], nil
}
