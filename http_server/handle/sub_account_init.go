package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
	"das_sub_account/tables"
	"fmt"
	"github.com/dotbitHQ/das-lib/common"
	"github.com/dotbitHQ/das-lib/core"
	"github.com/dotbitHQ/das-lib/molecule"
	"github.com/dotbitHQ/das-lib/smt"
	"github.com/dotbitHQ/das-lib/txbuilder"
	"github.com/dotbitHQ/das-lib/witness"
	"github.com/gin-gonic/gin"
	"github.com/nervosnetwork/ckb-sdk-go/indexer"
	"github.com/nervosnetwork/ckb-sdk-go/types"
	"github.com/scorpiotzh/toolib"
	"net/http"
	"strings"
	"sync"
)

type ReqSubAccountInit struct {
	core.ChainTypeAddress
	chainType common.ChainType
	address   string
	Account   string `json:"account"`
}

type RespSubAccountInit struct {
	SignInfoList
}

func (h *HttpHandle) SubAccountInit(ctx *gin.Context) {
	var (
		funcName               = "SubAccountInit"
		clientIp, remoteAddrIP = GetClientIp(ctx)
		req                    ReqSubAccountInit
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

	if err = h.doSubAccountInit(&req, &apiResp, clientIp, remoteAddrIP); err != nil {
		log.Error("doSubAccountInit err:", err.Error(), funcName, clientIp)
		doApiError(err, &apiResp)
	}

	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doSubAccountInit(req *ReqSubAccountInit, apiResp *api_code.ApiResp, clientIp, remoteAddrIP string) error {
	var resp RespSubAccountInit
	resp.List = make([]SignInfo, 0)

	// check params
	addrHex, err := req.FormatChainTypeAddress(config.Cfg.Server.Net, true)
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
	builder, err := h.DasCore.ConfigCellDataBuilderByTypeArgsList(common.ConfigCellTypeArgsSubAccount, common.ConfigCellTypeArgsAccount)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, err.Error())
		return fmt.Errorf("ConfigCellDataBuilderByTypeArgsList err: %s", err.Error())
	}

	subAccountBasicCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.BasicCapacity().RawData())
	subAccountPreparedFeeCapacity, _ := molecule.Bytes2GoU64(builder.ConfigCellSubAccount.PreparedFeeCapacity().RawData())
	subAccountCommonFee, _ := molecule.Bytes2GoU64(builder.ConfigCellAccount.CommonFee().RawData())

	log.Info("doSubAccountInit:", req.Account, acc.AccountId, clientIp, remoteAddrIP)
	capacityNeed, capacityForChange := subAccountBasicCapacity+subAccountPreparedFeeCapacity+subAccountCommonFee, common.DasLockWithBalanceTypeOccupiedCkb
	var liveCells []*indexer.LiveCell
	var change uint64
	var feeDasLock, feeDasType *types.Script

	feeDasLock, feeDasType, err = h.DasCore.Daf().HexToScript(*addrHex)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, fmt.Sprintf("HexToScript err: %s", err.Error()))
		return fmt.Errorf("HexToScript err: %s", err.Error())
	}
	total := uint64(0)
	liveCells, total, err = h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          h.DasCache,
		LockScript:        feeDasLock,
		CapacityNeed:      capacityNeed,
		CapacityForChange: capacityForChange,
		SearchOrder:       indexer.SearchOrderAsc,
	})
	if err != nil {
		return doDasBalanceError(err, apiResp)
	}
	change = total - capacityNeed

	// build tx
	buildParams := paramsSubAccountInitTx{
		req:                req,
		acc:                acc,
		liveCells:          liveCells,
		subAccountCapacity: subAccountBasicCapacity + subAccountPreparedFeeCapacity,
		txFee:              subAccountCommonFee,
		change:             change,
		feeDasLock:         feeDasLock,
		feeDasType:         feeDasType,
	}
	txParams, err := h.buildSubAccountInitTx(&buildParams)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "build tx params err: "+err.Error())
		return fmt.Errorf("buildSubAccountInitSubAccountTx err: %s", err.Error())
	}

	//txBuilder := txbuilder.NewDasTxBuilderFromBase(h.TxBuilderBase, nil)
	//if err := txBuilder.BuildTransaction(txParams); err != nil {
	//	apiResp.ApiRespErr(api_code.ApiCodeError500, "BuildTransaction err: "+err.Error())
	//	return fmt.Errorf("BuildTransaction err: %s", err.Error())
	//}

	signList, _, err := h.buildTx(&paramBuildTx{
		txParams:   txParams,
		skipGroups: []int{},
		chainType:  req.chainType,
		address:    req.address,
		action:     common.DasActionEnableSubAccount,
		account:    req.Account,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "buildTx err: "+err.Error())
		return fmt.Errorf("buildTx err: %s", err.Error())
	}

	resp.Action = common.DasActionEnableSubAccount
	resp.SignKey = signList.SignKey
	resp.List = signList.List

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
		Action:           common.DasActionEnableSubAccount,
		SmtRoot:          smt.H256Zero(),
		DasProfit:        0,
		OwnerProfit:      0,
		Flag:             witness.FlagTypeCustomRule,
		CustomScriptArgs: nil,
	}
	subAccountOutputData := witness.BuildSubAccountCellOutputData(subDataDetail)
	txParams.OutputsData = append(txParams.OutputsData, subAccountOutputData)
	if p.change > 0 {
		base := 1000 * common.OneCkb
		splitList, err := core.SplitOutputCell2(p.change, base, 10, p.feeDasLock, p.feeDasType, indexer.SearchOrderDesc)
		if err != nil {
			return nil, fmt.Errorf("SplitOutputCell2 err: %s", err.Error())
		}
		for i := range splitList {
			txParams.Outputs = append(txParams.Outputs, splitList[i])
			txParams.OutputsData = append(txParams.OutputsData, []byte{})
		}

		//txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		//	Capacity: p.change,
		//	Lock:     p.feeDasLock,
		//	Type:     p.feeDasType,
		//})
		//txParams.OutputsData = append(txParams.OutputsData, []byte{})
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
	)

	return &txParams, nil
}

func (h *HttpHandle) getSvrBalance(p paramBalance) (uint64, []*indexer.LiveCell, error) {
	if p.capacityForNeed == 0 {
		return 0, nil, fmt.Errorf("needCapacity is nil")
	}
	svrBalanceLock.Lock()
	defer svrBalanceLock.Unlock()

	liveCells, total, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:           h.DasCache,
		LockScript:         p.svrLock,
		CapacityNeed:       p.capacityForNeed,
		CurrentBlockNumber: 0,
		CapacityForChange:  common.MinCellOccupiedCkb,
		SearchOrder:        indexer.SearchOrderDesc,
	})
	if err != nil {
		return 0, nil, fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}

	var outpoints []string
	for _, v := range liveCells {
		outpoints = append(outpoints, common.OutPointStruct2String(v.OutPoint))
	}
	h.DasCache.AddOutPoint(outpoints)

	return total - p.capacityForNeed, liveCells, nil
}

type paramBalance struct {
	svrLock           *types.Script
	capacityForNeed   uint64
	capacityForChange uint64
}

var svrBalanceLock sync.Mutex
