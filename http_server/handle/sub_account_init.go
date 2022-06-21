package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/DeAccountSystems/das-lib/common"
	"github.com/DeAccountSystems/das-lib/core"
	"github.com/DeAccountSystems/das-lib/molecule"
	"github.com/DeAccountSystems/das-lib/smt"
	"github.com/DeAccountSystems/das-lib/txbuilder"
	"github.com/DeAccountSystems/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
)

type ReqSubAccountInit struct {
	api_code.ChainTypeAddress
	chainType common.ChainType
	address   string
	Account   string `json:"account"`
}

type RespSubAccountInit struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountInit(ctx *gin.Context) {
	var (
		funcName = "SubAccountInit"
		clientIp = GetClientIp(ctx)
		req      ReqSubAccountInit
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

	if err = h.doSubAccountInit(&req, &apiResp); err != nil {
		log.Error("doSubAccountInit err:", err.Error(), funcName, clientIp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountInit(req *ReqSubAccountInit, apiResp *api_code.ApiResp) error {
	var resp RespSubAccountInit
	resp.List = make([]SignInfo, 0)

	// check params
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params is invalid: "+err.Error())
		return nil
	}
	req.chainType, req.address = addrHex.ChainType, addrHex.AddressHex

	// check update
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}

	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}

	// check account
	acc, err := h.doSubAccountCheckAccount(req.Account, apiResp, common.DasActionEnableSubAccount)
	if err != nil {
		return fmt.Errorf("doSubAccountCheckAccount err: %s", err.Error())
	} else if apiResp.ErrNo != api_code.ApiCodeSuccess {
		return nil
	}
	if acc.OwnerChainType != req.chainType || !strings.EqualFold(acc.Owner, req.address) {
		apiResp.ApiRespErr(api_code.ApiCodePermissionDenied, "not have owner permission")
		return nil
	}

	// config cell
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount, common.ConfigCellTypeArgsAccount, common.ConfigCellTypeArgsSubAccountWhiteList)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}

	// check white list
	if builder.ConfigCellSubAccountWhiteListMap == nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "white list error")
		return fmt.Errorf("ConfigCellSubAccountWhiteListMap is nil")
	}
	isAllOpen := false
	if _, ok := builder.ConfigCellSubAccountWhiteListMap["0xd83bc404a35ee0c4c2055d5ac13a5c323aae494a"]; ok {
		isAllOpen = true
	}
	log.Info("doSubAccountInit:", req.Account, isAllOpen)
	if !isAllOpen {
		if _, ok := builder.ConfigCellSubAccountWhiteListMap[acc.AccountId]; !ok {
			apiResp.ApiRespErr(api_code.ApiCodeUnableInit, fmt.Sprintf("account [%s] unable init", req.Account))
			return nil
		}
	}

	subAccountBasicCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	subAccountPreparedFeeCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	subAccountCommonFee, _ := molecule.Bytes2GoU64(builder.ConfigCellAccount.CommonFee().RawData())

	// check balance
	dasLock, dasType, err := h.DasCore.Daf().HexToScript(*addrHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToScript err: %s", err.Error()))
		return fmt.Errorf("HexToScript err: %s", err.Error())
	}
	capacityNeed, capacityForChange := subAccountBasicCapacity+subAccountPreparedFeeCapacity+subAccountCommonFee, common.DasLockWithBalanceTypeOccupiedCkb
	liveCells, total, err := core.GetSatisfiedCapacityLiveCellWithOrder(h.DasCore.Client(), h.DasCache, dasLock, dasType, capacityNeed, capacityForChange, indexer.SearchOrderAsc)
	if err != nil {
		return doDasBalanceError(err, apiResp)
	}

	// build tx
	buildParams := paramsSubAccountInitTx{
		req:                req,
		acc:                acc,
		liveCells:          liveCells,
		subAccountCapacity: subAccountBasicCapacity + subAccountPreparedFeeCapacity,
		txFee:              subAccountCommonFee,
		change:             total - subAccountBasicCapacity - subAccountPreparedFeeCapacity - subAccountCommonFee,
		feeDasLock:         dasLock,
		feeDasType:         dasType,
	}
	txParams, err := h.buildSubAccountInitTx(&buildParams)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx params err: "+err.Error())
		return fmt.Errorf("buildSubAccountInitSubAccountTx err: %s", err.Error())
	}

	txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	if err := txBuilder.BuildTransaction(txParams); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "BuildTransaction err: "+err.Error())
		return fmt.Errorf("BuildTransaction err: %s", err.Error())
	}

	signList, err := txBuilder.GenerateDigestListFromTx([]int{})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "GenerateDigestListFromTx err: "+err.Error())
		return fmt.Errorf("GenerateDigestListFromTx err: %s", err.Error())
	}

	log.Info("buildTx:", txBuilder.TxString())

	// cache
	sic := SignInfoCache{
		ChainType: req.chainType,
		Address:   req.address,
		Action:    common.DasActionEnableSubAccount,
		Account:   req.Account,
		Capacity:  0,
		BuilderTx: nil,
	}
	sic.BuilderTx = txBuilder.DasTxBuilderTransaction
	signKey := sic.SignKey()
	cacheStr := toolib.JsonString(&sic)
	if err = h.RC.SetSignTxCache(signKey, cacheStr); err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "SetSignTxCache err: "+err.Error())
		return fmt.Errorf("SetSignTxCache err: %s", err.Error())
	}

	resp.Action = common.DasActionEnableSubAccount
	resp.SignKey = signKey
	resp.List = append(resp.List, SignInfo{
		//SignKey:  "",
		SignList: signList,
	})

	apiResp.ApiRespOK(resp)
	return nil
}

type paramsSubAccountInitTx struct {
	req                *ReqSubAccountInit
	acc                *tables.TableAccountInfo
	liveCells          []*indexer.LiveCell
	subAccountCapacity uint64
	txFee              uint64
	change             uint64
	feeDasLock         *types.Script
	feeDasType         *types.Script
}

func (h *HttpHandle) buildSubAccountInitTx(p *paramsSubAccountInitTx) (*txbuilder.BuildTransactionParams, error) {
	var txParams txbuilder.BuildTransactionParams
	// inputs
	outpoint := common.String2OutPointStruct(p.acc.Outpoint)
	txParams.Inputs = append(txParams.Inputs, &types.CellInput{
		PreviousOutput: outpoint,
	})

	for _, v := range p.liveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// outputs
	contractDas, err := core.GetDasContractInfo(common.DasContractNameDispatchCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractAccount, err := core.GetDasContractInfo(common.DasContractNameAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractAlwaysSuccess, err := core.GetDasContractInfo(common.DasContractNameAlwaysSuccess)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	contractSubAccount, err := core.GetDasContractInfo(common.DASContractNameSubAccountCellType)
	if err != nil {
		return nil, fmt.Errorf("GetDasContractInfo err: %s", err.Error())
	}
	res, err := h.DasCore.Client().GetTransaction(h.Ctx, outpoint.TxHash)
	if err != nil {
		return nil, fmt.Errorf("GetTransaction err: %s", err.Error())
	}

	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: res.Transaction.Outputs[outpoint.Index].Capacity,
		Lock:     res.Transaction.Outputs[outpoint.Index].Lock,
		Type:     res.Transaction.Outputs[outpoint.Index].Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, res.Transaction.OutputsData[outpoint.Index])
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: p.subAccountCapacity,
		Lock:     contractAlwaysSuccess.ToScript(nil),
		Type:     contractSubAccount.ToScript(common.Hex2Bytes(p.acc.AccountId)),
	})
	subDataDetail := witness.SubAccountCellDataDetail{
		SmtRoot:    smt.H256Zero(),
		DasProfit:  0,
		HashType:   nil,
		CustomArgs: nil,
	}
	subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, subAccountOutputData)
	if p.change > 0 {
		txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
			Capacity: p.change,
			Lock:     p.feeDasLock,
			Type:     p.feeDasType,
		})
		txParams.OutputsData = append(txParams.OutputsData, []byte{})
	}

	// witness
	actionWitness, err := witness.GenActionDataWitness(common.DasActionEnableSubAccount, common.Hex2Bytes(common.ParamOwner))
	if err != nil {
		return nil, fmt.Errorf("GenActionDataWitness err: %s", err.Error())
	}
	txParams.Witnesses = append(txParams.Witnesses, actionWitness)

	builderMap, err := witness.AccountIdCellDataBuilderFromTx(res.Transaction, common.DataTypeNew)
	if err != nil {
		return nil, fmt.Errorf("AccountIdCellDataBuilderFromTx err: %s", err.Error())
	}
	builder, ok := builderMap[p.acc.AccountId]
	if !ok {
		return nil, fmt.Errorf("builderMap not exist account: %s", p.acc.Account)
	}
	accWitness, accData, err := builder.GenWitness(&witness.AccountCellParam{
		OldIndex:             0,
		NewIndex:             0,
		Action:               common.DasActionEnableSubAccount,
		EnableSubAccount:     tables.AccountEnableStatusOn,
		RenewSubAccountPrice: common.OneCkb,
	})
	accData = append(accData, res.Transaction.OutputsData[builder.Index][32:]...)
	txParams.OutputsData[0] = accData
	txParams.Witnesses = append(txParams.Witnesses, accWitness)

	// cell deps
	configCellMain, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsMain)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	configCellAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	configCellSubAcc, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccount)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	heightCell, err := h.DasCore.GetHeightCell()
	if err != nil {
		return nil, fmt.Errorf("GetHeightCell err: %s", err.Error())
	}
	timeCell, err := h.DasCore.GetTimeCell()
	if err != nil {
		return nil, fmt.Errorf("GetTimeCell err: %s", err.Error())
	}
	configCellWhiteList, err := core.GetDasConfigCellInfo(common.ConfigCellTypeArgsSubAccountWhiteList)
	if err != nil {
		return nil, fmt.Errorf("GetDasConfigCellInfo err: %s", err.Error())
	}
	txParams.CellDeps = append(txParams.CellDeps,
		configCellMain.ToCellDep(),
		contractDas.ToCellDep(),
		contractAlwaysSuccess.ToCellDep(),
		contractSubAccount.ToCellDep(),
		contractAccount.ToCellDep(),
		configCellAcc.ToCellDep(),
		configCellSubAcc.ToCellDep(),
		heightCell.ToCellDep(),
		timeCell.ToCellDep(),
		configCellWhiteList.ToCellDep(),
	)

	return &txParams, nil
}
