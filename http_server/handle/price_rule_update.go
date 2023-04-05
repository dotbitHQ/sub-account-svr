package handle

import (
	"das_sub_account/config"
	"das_sub_account/http_server/api_code"
	"das_sub_account/internal"
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
)

type ReqPriceRuleUpdate struct {
	core.ChainTypeAddress
	Account string `json:"account" binding:"required"`
}

func (h *HttpHandle) PriceRuleUpdate(ctx *gin.Context) {
	var (
		funcName = "PriceRuleUpdate"
		clientIp = GetClientIp(ctx)
		req      ReqPriceRuleUpdate
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

	if err = h.doPriceRuleUpdate(&req, &apiResp); err != nil {
		log.Error("doConfigAutoMintUpdate err:", err.Error(), funcName, clientIp)
	}
	ctx.JSON(http.StatusOK, apiResp)
}

func (h *HttpHandle) doPriceRuleUpdate(req *ReqPriceRuleUpdate, apiResp *api_code.ApiResp) error {
	if err := h.checkSystemUpgrade(apiResp); err != nil {
		return fmt.Errorf("checkSystemUpgrade err: %s", err.Error())
	}
	if ok := internal.IsLatestBlockNumber(config.Cfg.Server.ParserUrl); !ok {
		apiResp.ApiRespErr(api_code.ApiCodeSyncBlockNumber, "sync block number")
		return fmt.Errorf("sync block number")
	}
	res, err := req.ChainTypeAddress.FormatChainTypeAddress(h.DasCore.NetType(), true)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeParamsInvalid, "params invalid")
		return err
	}
	address := common.FormatAddressPayload(res.AddressPayload, res.DasAlgorithmId)

	if err := h.checkAuth(address, req.Account, apiResp); err != nil {
		return err
	}
	parentAccountId := common.Bytes2Hex(common.GetAccountIdByAccount(req.Account))

	baseInfo, err := h.TxTool.GetBaseInfo()
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "server error")
		return err
	}

	accountCell, err := h.getAccountOrSubAccountCell(baseInfo.ContractAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	accountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, accountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}
	subAccountCell, err := h.getAccountOrSubAccountCell(baseInfo.ContractAcc, parentAccountId)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("getAccountOrSubAccountCell err: %s", err.Error())
	}
	subAccountTx, err := h.DasCore.Client().GetTransaction(h.Ctx, subAccountCell.OutPoint.TxHash)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return err
	}

	txParams := &txbuilder.BuildTransactionParams{}
	txParams.CellDeps = append(txParams.CellDeps,
		baseInfo.ContractAcc.ToCellDep(),
		baseInfo.ContractSubAcc.ToCellDep(),
		baseInfo.TimeCell.ToCellDep(),
		baseInfo.HeightCell.ToCellDep(),
		baseInfo.ConfigCellAcc.ToCellDep(),
		baseInfo.ConfigCellSubAcc.ToCellDep(),
	)

	dasLock, _, err := h.DasCore.Daf().HexToScript(*res)
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "HexToArgs err")
		return err
	}
	balanceLiveCells, _, err := h.DasCore.GetBalanceCells(&core.ParamGetBalanceCells{
		DasCache:          h.DasCache,
		LockScript:        dasLock,
		CapacityNeed:      common.OneCkb,
		CapacityForChange: common.DasLockWithBalanceTypeOccupiedCkb,
		SearchOrder:       indexer.SearchOrderDesc,
	})
	if err != nil {
		apiResp.ApiRespErr(api_code.ApiCodeError500, "internal error")
		return fmt.Errorf("GetBalanceCells err: %s", err.Error())
	}
	txParams.Inputs = append(txParams.Inputs,
		&types.CellInput{
			PreviousOutput: accountCell.OutPoint,
		},
		&types.CellInput{
			PreviousOutput: subAccountCell.OutPoint,
		},
	)
	for _, v := range balanceLiveCells {
		txParams.Inputs = append(txParams.Inputs, &types.CellInput{
			PreviousOutput: v.OutPoint,
		})
	}

	// account cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: accountTx.Transaction.Outputs[accountCell.TxIndex].Capacity,
		Lock:     accountTx.Transaction.Outputs[accountCell.TxIndex].Lock,
		Type:     accountTx.Transaction.Outputs[accountCell.TxIndex].Type,
	})
	txParams.OutputsData = append(txParams.OutputsData, accountTx.Transaction.OutputsData[accountCell.TxIndex])

	// sub_account cell
	txParams.Outputs = append(txParams.Outputs, &types.CellOutput{
		Capacity: subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Capacity,
		Lock:     subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Lock,
		Type:     subAccountTx.Transaction.Outputs[subAccountCell.TxIndex].Type,
	})
	subAccountCellDetail := witness.ConvertSubAccountCellOutputData(subAccountTx.Transaction.OutputsData[subAccountCell.TxIndex])
	subAccountCellDetail.Flag = witness.FlagTypeCustomRule
	subAccountCellDetail.AutoDistribution = witness.AutoDistributionEnable

	// TODO PriceRulesHash/PreservedRulesHash

	newSubAccountCellOutputData := witness.BuildSubAccountCellOutputData(subAccountCellDetail)
	txParams.OutputsData = append(txParams.OutputsData, newSubAccountCellOutputData)

	for _, v := range balanceLiveCells {
		txParams.Outputs = append(txParams.Outputs, v.Output)
		txParams.OutputsData = append(txParams.OutputsData, v.OutputData)
	}
	return nil
}
